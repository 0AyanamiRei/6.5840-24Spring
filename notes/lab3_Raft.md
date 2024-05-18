# Raft

翻译论文笔者觉得没有太大的意义了, 网上能找到很多关于**Raft**原理的文章, 所以笔者这里一边总结**Raft**(主要面向自己日后回顾), 一边过一下自己关于实验的设计, 这里是一些关于**Raft**的文章:

## Interface: API for k/v server

实际应用中, *raft*以*lib*的形式存在, 假设我们使用的是*k/v server*, 那么每个服务将有两部分代码组成:**应用程序代码和Raft库**(见lec课上画的图), 因此*raft lib*需要提供接口给应用程序:

1. `Make(...)`: 创建一个新的*Raft*服务器
2. `Start(...)`: 应用程序告诉*raft*记录一个*log*
3. `applyCh <- ApplyMsg`: *raft*需要告诉应用程序该*log*已写好了(同时还需要通知其他*followers*)
4. `GetState()`: 询问当前*term*, 以及该服务器是不是*leader*

***1.Make()***

```go
// peers      []*labrpc.ClientEnd
// me         int
// persister  *Persister
// applyCh    chan ApplyMsg

func Make(peers, me, persister, applyCh) *Raft {
	rf := &Raft{}
    ....
	return rf
}
```

应用程序将所有*raft server*的网络标识符`peers`告诉新建的服务器, 每个*raft*都保留该状态以便互相通信; `peers[me]`则是该服务器对应的标识符; `persister`告诉服务器应该把需要持久化的状态存放在哪儿; `applyCh`则提供了服务器向应用程序发送*ApplyMsg*的通道, 前面也提到了是为了通知应用程序写日志结束.

***2.Start()***

```go
// command   interface{}

func (rf *Raft) Start(command) (int, int, bool) {
    ....
	return index, term, isLeader
}
```

应用程序将要追加到日志的命令`command`发送给服务器, 返回该命令在日志中存放的位置以及当前的任期号和该服务器是否为*leader*

## Leader election

1. **server state**: *leader*, *candidate*, *follower*
2. **election timeout**
3. **RPCs**: *RequestVote*

对于一个*raft*服务器, 如果在一个给定时间:`election timeout`内未收到来自当前任期下的*leader*的心跳消息(实质是没有附带日志内容的`AppendEntries RPC`)就会认为该*leader*挂了, 那么会重新发起一轮投票, 具体如下:

```go
// 发起本轮竞选的服务器
rf.CurrentTerm ++
change state:  follow->candidate
vote self
in parallel:   rf.peers[0~n].Call("Raft.RequestVote", args, reply)

// 其余服务器回应该rpc
peers[0] call: RequestVote(...)
peers[1] call: RequestVote(...)
...
peers[n] call: RequestVote(...)
```

### leadr问题记录

1. 如果某个term比较落后的候选人得到了一个带有更新的term的回复, 那是继续这轮请求还是更新term后退出这轮请求?
2. 发起一轮投票后, 该怎么实现论文里说的三种结果 `a:赢得竞选`, `b:其他候选人获胜`, `c:平票, 重新来一轮`, 主要问题在如何确认"其他候选人获胜"这一消息来停止这轮投票, 感觉会用到条件变量之类的.

```
2024/5/15 0:10  锦林哥哥断火了, 我想死, 当时奈何有了前车之鉴, 我是个胆小鬼, 死不成还让学校发现了差点休学了,所以找了个理由活着：未来尽义务赡养父母

2024/5/15 10:54 锦哥哥理我了, 但是我现在什么都不想做, 就这样吧.

具体设计明日再谈, 大概先做这些事:

1. 先把论文关于"选举"+"心跳"部分的内容重温一遍
2. 把hints看完, 总结一下
```
### 论文内容

关于`term`过时的*leader*和*candidate*, 会立刻降为*follower*, 不再进行任何*leader*和*candidate*的行为, 同时服务器会拒掉来自过时服务器的任何请求.

服务器会重试`rpc`, 如果它们没有收到回复

需要注意的是, *candidate*发投票时的rpc和*leader*当选时发的心跳消息论文里提到是并行发的, 如果是用`for...`这样的形式, 下一条消息要等待上一个结束猜测会有问题. 

关于每个候选人结束此轮投票的三种情况:

- 1, 自己获胜
- 2, 其他候选人获胜
- 3, 平票

### hint内容

1. 关于心跳消息频率的要求, hints里说明了每秒不超过10次
2. 测试要求5秒内必须选出新的*leader*(如果大多数服务器还正常)
3. 论文中提到的选举超时设置为`150~300ms`, 这只适合心跳频率远远大于`150ms/times`, 比如`10ms/times`才合适, 由于测试限制了每秒的心跳次数, 所以你需要仔细设置超时时间.
4. lec里Morris提到了, 选举超时的下限最好是心跳周期的倍数, 上限取决于需要多高的性能, 另外一个问题是不同节点选举定时器的超时时间差要足够长, 至少要大于完成一轮选举的时间, 最少就是发送一条RPC的往返时间.

### Some Bug


第一个遇到的问题是由于只关注leader的任期过期更新, 忽略了让follower收到更新的任期后也更新.

- 这个问题比较好解决, 严格按照原则`每轮rpc, 发送和回复都附带任期, 更新较小服务器的任期且设置状态为FOLLOWER`即可避免这样的犯错


第二个遇到的问题是记录服务器的"已投票"状态, 如果只记录投给了谁, 那么难以在下一轮投票之前清空该记录来进行下一轮的投票, 实际上我遇到的问题就是在第一轮投票结束后一段时间, 让领导人挂掉, 然后选举超时剩下的服务器进行第二轮投票, 发现第一轮记录的状态没有被清理

- 思考后, 我觉得在收到心跳消息时清空该状态不错, 但是转头一想, 平票时第二轮选举就有问题, 因为连续两次投票之间没有心跳的环节, 所以我的决定是记录下<任期, 投票ID>, 不必清空服务器投给了谁这个信息, 每次检查两个状态即可(不过我感觉只需要记录投票任期即可)

如果采用for循环一个一个等待投票rpc返回就会导致这样的问题(虽然早就知道了,不过当时不会实现`得到大多数投票就成为leader`的机制)
```
p0(C)-2-2.0554107s: Vote Start (t2)
time: 0s p0->p1
time: 0s p0->p2
p1(F)-2-2.0548926s: Voted p1->p0
(855.2739ms, 也就是说rpc超时的时间大概就是这么久)
time: 855.2739ms  p0->p2 failed
p0(C)-2-2.9106846s: Vote End with 2votes (t2)
   
p1(C)-3- 2.351764s: Vote Start (t3)
          time: 0s p1->p0
```

一个死锁的例子, 原因是: *p1发*起投票后在等待*rpc*超时, 此时*p2*选举超时同时开启了投票, 由于发起投票的时候就要获取自身的锁, 所以当*p1*收到*rpc*超时消息后会先询问*p2*投票, 在*p2*的*RequestVote*函数中等待*p2*的锁, 然后*p2*在自己发起的投票中又会询问*p1*, 同样地在*RequestVote*函数中等待*p1*的锁, 这个时候就死锁了

所以用`for...`依次请求投票不是一个好主意, 更何况后面还会有网络延迟的模拟, 这样会导致等待时间非常的长.

```
原来的leader p0挂了后重新选举的结果
0 died
22:43:25.259333 p1(C)-2: Vote Start (t2)←-------------+ p1.mu.lock()
22:43:25.259333 p1->p0                                |
22:43:25.342724 p2(C)-2: Vote Start (t2)←-------------|----+ p2.mu.lock()
22:43:25.342724 p2->p0                                |    |
22:43:25.344264 p1->p0 failed                         |    |
22:43:25.344264 p1->p2--------------------------------|----|-→wait p2.mu.lock()
                                                      |    |
22:43:28.281585 p2->p0 failed                         |    |
22:43:28.281585 p2->p1--------------------------------|----|-→wait p1.mu.lock()
                                                      |    |
```

修改成并发调用后, ticker()线程中我们只需要调用一下sendRequestVote(), 创建多个线程向其他服务器请求投票, 然后就可以从函数中返回, 或者说等待这些投票rpc返回处理后再从函数中返回. 如果直接从函数中返回, 那么如何处理平票是一个问题, 因为我们只能在每个请求投票线程中判断是否得到大多数服务器投票, 然后修改状态+发送心跳, 但如果vote数量一直不大于服务器总数的一半, 也就是最后得到的结果是竞选失败,我们需要去处理这个问题, 这里观察了一下代码:

目前我没有发现CANDIDATE这个服务器状态有任何作用, 因为所以我们暂且不设置该状态, 就只设置leader和follower两个状态

### DEBUG_printf策略

先列举一下我们用到的函数(线程)

1. 每个服务器开了三个线程, 分别用于rpc收发信息, 心跳机制, 超时选举
2. *leader*发送心跳消息的函数`sendHeartbeat()`, 其他服务器接收并处理心跳消息的函数`Heartbeat()`
3. *candidate*发送选举消息的函数`sendRequestVote()`, 其他服务器进行投票的函数`RequestVote()`

调试详细的格式按照:`p[i]-t-TFZ: xxxx`, `p[i]`是事件发起者的服务器标识, `t`是其任期, `TFZ`则是自己实现的从该服务器创建时就开始计时的计时器, 后面的`xxxx`是具体要展示的信息, 我们有以下几种信息按规定输出:

**对每一轮选举的调试信息**: 发起选举的*candidate*需要展示`Vote Start`和`Vote End`这两个信息, 并且在这一轮投票结束后输出其得票数, 投票人也需要展示自己投给的目标, 后续可能需要有拒绝投票的理由, `%s`表示的是服务器的状态: `C`, `L`, `F`

```go
if DEBUG_Vote{
	fmt.Printf("p%d(%s)-%d-%10v: Vote Start (t%d)\n")
}

if DEBUG_Vote{
	fmt.Printf("p%d(%s)-%d-%10v: Vote End with %dvotes (t%d)\n")
}


if DEBUG_Vote{
	fmt.Printf("p%d(%s)-%d-%10v: Voted p%d->p%d\n")
}

if DEBUG_Vote{
	fmt.Printf("p%d(%s)-%d-%10v: Refuse, had vote p%d->p%d\n")
}
```

**对心跳消息的调试信息**: 发出心跳消息的只能是leader, 所以只需要按常规信息输出即可

```go
if DEBUG_Heartbeat{
	fmt.Printf("p%d(%s)-%d-%10v: !!!Heartbeat!!!\n")
}

if DEBUG_Heartbeat{
	fmt.Printf("p%d(%s)-%d-%10v: p%d(L)->p%d(F) beat!!!\n")
}

```


## Log replicated

第一个需要回答的问题是`raft如何保证leader拥有所有committed log entries`

论文中强调了*raft*的*rpcs*只会由*leader*发出(非选举), 所以我们要回答的是在这种情况下如何保持上述的性质的, 这一点在*raft*中通过选举制度来保证的, 以下是我的思考, 主要利用的性质还是两次**majority**必有重叠

```
1. 由于需要大多数followers写入日志才能提交, 所有对某一个committed log来说, 持有它的服务器数量一定大于N/2

2. 当选leader需要获得大多数的票数, 且仅持有更新的日志才会得票, 所以当选leader比超过N/2的服务器更新

3. 由于两次majority必定有重叠部分, 所以当选的leader必定持有该committed log

4. 推广该committed log, 那么对每个committed log, 该当选leader必定持有

这里隐藏了一个事实, 是论文提到的在AppendEntries RPC时进行一次简单的一致性检验保证的事实: 如果两个log entries在两个服务器日志中的相同位置, 且任期一致, 那么可以断言这两个服务器的在此之前的日志全部相等
```