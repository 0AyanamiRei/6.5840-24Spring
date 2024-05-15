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
