# Raft

[分布式系统中的网络模型和故障模型](https://danielw.cn/network-failure-models)
[非对称网络分区](https://link.zhihu.com/?target=https%3A//github.com/baidu/braft/blob/master/docs/cn/raft_protocol.md%23symmetric-network-partitioning)

这里记录的是Raft作者博士论文里提到的一些优化, 以及个人在写实验时收集到的一些问题和针对该问题做的思考.

## 问题一: 网络分区导致的脑裂

这延续了*Raft*使用投票的方式解决**脑裂**这一个事实, 以一个5节点的集群为例子, 假设这5个节点命令如下, 括号内是它当前的状态: 

- 领导人: `A(L,1)`
- 追随者: `B(F,1)`, `C(F,1)`, `D(F,1)`, `E(F,1)`

某个时刻, 发生了网络分区的故障, 使得集群划分为如下的分区, 不同分区的服务器不能互相通信.

- 分区1: `A(L,1)`, `B(F,1)`, `C(F,1)`
- 分区2: `D(F,1)`, `E(F,1)`

得益于领导人的选拔机制, 分区2中的两个服务器即使在选举超时后发起一轮投票也无法上任领导人.  但这只是一个很朴素的网络分区故障, 接着看另一个分区情况:

- 分区1: `A(L,1)`, `B(F,1)`            ->`A(L,1)`, `B(F,1)` 
- 分区2: `C(F,1)`, `D(F,1)`, `E(F,1)`  ->`C(L,2)`, `D(F,2)`, `E(F,2)`

如果是领导人被割裂到了拥有较少服务器的分区, 那么另一个分区因拥有较多服务器, 所以会在新的一轮投票中产生新的领导人, 这个时候就发生了脑裂的问题---**分区1,2都有各自的领导人**

在实验的测试代码`cfg.one(...)`中我们可以看到对于客户端发送的请求, 采取的是轮询的方式找到领导人, 虽然在此次实验中我们提供给客户端的接口`Start(command interface{})`明确地注释了不保证该请求一定被提交和应用.  因为在测试代码中我们可以选择`cfg.one(...retry=true)`这种方式, 即一段时间内未观察到该请求被应用则接着上一次发往服务器的顺序往下轮询---直到超过设置的时间或发往了真正有用的领导人, 也就是分区2的`C(L,2)`.

***笔者想到的解决方案***

1. 租约
2. 间隔某几次心跳包就做一次虚假投票

## 问题二: 网络分区导致任期快速增长(对应论文9.6)

这个问题其实笔者在查日志的时候发现了, 但是当时并没有把它认定是一个很严重的问题( *毕竟是在这种实验环境下,以给出的测试为参考, 对*Raft*的实现没有那么严苛* ), 在*Raft*大论文里面既然作者提到了, 那我便列在这里了.

仍然是网络分区这个故障, 使我们得到以下的集群划分状况:

- 分区1: `A(L,1)`, `B(F,1)`, `C(F,1)`
- 分区2: `D(F,1)`, `E(F,1)`  ->. . . . . .->`D(F,10)`, `E(F,10)`

分区2的两个服务器既无法收到领导人的心跳消息, 也无法获得大多数服务器投票自己上任, 导致了其不停地发起选举, 增加任期, 一旦网络的故障修复, 领导人`A(L,1)`就会立刻下台, 重新选举.

硬要找出点什么影响的话, 我想就是: `分区2的服务器浪费了不必要的网络资源`, `网络修复后扰乱了一次领导人身份`, `任期数过大`.

***Raft作者给出的解决方案***

## 问题三: 非对称的网络分区导致集群领导人反复下台

**这个问题可以在Braft的文档里找到, 更完整的内容还请阅读文档**
[非对称网络分区](https://link.zhihu.com/?target=https%3A//github.com/baidu/braft/blob/master/docs/cn/raft_protocol.md%23symmetric-network-partitioning)

我们思考网络分区导致的故障时, 没有考虑过某一个服务器属于多个分区的情况:

- 分区1: `S1(L,1)`, `S3(F,1)`
- 分区2: `S2(F,1)`, `S3(F,1)`

这里的问题是, `S1`无法收到领导人的心跳, 自己不停地增加任期发起选举导致`S3`的任期也不断增加, 使得领导人下台, 如果下一次选出来的领导人还是`S2`, 那么这个问题还会持续发生.

你可能会想, 如果采纳了问题二的解决方案*PreVote*, 不就是可以避免掉`S1`不断自增任期发起选举吗? 

[这里](https://www.zhihu.com/question/483967518)答主给出了一个分区的情况(借鉴一下答主的图):

![pic](/6.5840-24Spring/pic/不对称分区的故障.webp)

很庆幸, 如果我们解决了问题一, 那么这里离群的领导人`4`就会下台. 不过在思考这个问题的时候, 又引发了对*PreVote*细节的思考, 以这里答主的回复来看, *PreVote*只会被同样选举超时的服务器同意, 这是否是个不必要的条件? 把*PreVote*单纯的视为一个自身服务器网络状况的检查是否更好?





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

目前我没有发现CANDIDATE这个服务器状态有任何作用, 所以我们暂且不设置该状态, 就只设置leader和follower两个状态 

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

**python dtest.py -n 10 -p 5 -s -v 3A** 使用助教写的脚本测试

## Log replicated

为了便于实验的进行, 重新读一遍论文这部分内容.

从客户端发出一个指令`SET X 8`, 到看到这条指令执行后的状态, 如果全程正常, 那么会到如下过程:

1. 客户端将指令发到leader(**Q1**)
2. leader将该指令写到自己的log中, 并并行发起rpc给其他followers
3. follower收到rpc后写到自己的log中, leader观察到大多数followers都写入log后, apply这条指令
4. 客户端能够看到该指令执行后的结果

这4个流程结束后, 现在整个raft的状态是所有服务器均把该指令写入到各自的log中, 但是只有在leader中能够观察到X=8.  在leader发送的下一条rpc消息中(**Q2**), 会告诉follower`SET X 8`这条指令已经是committed的, 然后follower便apply这条指令到其自己的状态机中.

最终`SET X 8`这条指令以及`X=8`这个状态会在所有服务器中看到, 也就是最终一致性.

那么如果是出现了论文中figure7那种严重的不一致情况呢, raft给出的解决方案是强制所有服务器保持与leader的一致, 也就是说leader会将与其log不一致的服务器的log修正来与其一致(**Q3**)

所以现在让我来简单总结一下日志复制leader需要做的事情, 并给出一些伪代码:

在此之前再来看一眼**Log Matching Property**: 

**log1[i].Term==log2[i].Term则log1[i]==log2[i]**

**log1[i]==log2[i], 则log1[0~i]==log2[0~i]**

1. AppendEntries 消息中, leader会和follower做一个简单的一致性检查, 也就是结构体中的`PrevLogIndex`和`PrevLogTerm`两个部分
2. 我决定采用论文中提到的减少rpcs次数的优化; 如果前面提到的一致性检查不一致, 那么follower将返回一个下标`i`给leader, 这样下次leader会将`log[i~..]`的日志内容发给它(**Q4**)
3. 每个服务器记录自己日志中已提交的下标, leader不会主动提交一个以前的leader留下的日志, 比如在`CommitIndex=2`, 而当前已有日志`log[1~5]`且`log[5]`为当前任期写入时, leader会等待`log[5]`被大多数服务器写入日志, 然后把`log[3~5]`一起提交后设置`CommitIndex=5`再告诉其他服务器提交`log[3~5]`



```
Q1: 如何做到将请求发给可能会更改的leader的?
Q2: 论文中提到了"retries AppendEntries RPC", 但是心跳消息会不断的发送, 是否可以不必重试某一次失败的RPC?
Q3: 这是否会有这样一种风险, leader自己因为故障没有某个已提交的log, 但是由于强制更新, 使其他服务器中丢掉该已提交的log?
Q4: 如何设置这个算法?
```

**Q1:**

**Q2:**

**Q4:**

```
index     1      2      3      4      5        6 
leader:  t=1 -> t=1 -> t=2 -> t=2 -> t=2 -> | t=3 
S1:      t=1 -> t=1 -> t=2
S2:      t=1 -> t=2

对于S1:
L->S1: AE rpc {log[6]; PrevLogIndex=5,PrevLogTerm=2}
S1->L: AE rpc {Success=false; Index=4}
L->S1: HT rpc {log[4~6]; PrevLogIndex=3,PrevLogTerm=2}
S1->L: HT rpc {Success=true}

对于S2:
L->S1: AE rpc {log[6]; PrevLogIndex=5,PrevLogTerm=2}
S1->L: AE rpc {Success=false; Index=3}
L->S1: HT rpc {log[3~6]; PrevLogIndex=2,PrevLogTerm=1}
S1->L: AE rpc {Success=false; Index=2}
L->S1: HT rpc {log[2~6]; PrevLogIndex=1,PrevLogTerm=1}
S1->L: AE rpc {Success=true}

2024/5/19 BUG lec里Morris给出了他的解决方案, 让我找到我给出的方案的BUG
index     1      2      3         4      
leader:  t=4 -> t=6 -> t=6 -> |  t=6
S :      t=4 -> t=5 -> t=5

L->S: AE rpc {log[4]; PrevLogIndex=3,PrevLogTerm=6}
S->L: AE rpc {Success=false; Index=4}
L->S: AE rpc {log[4]; PrevLogIndex=3,PrevLogTerm=6}
S->L: AE rpc {Success=false; Index=4}
```

对于一致性检查不过的服务器, 找到日志中满足*term*=*PrevLogTerm*的log或者是*term*<*PrevLogTerm*的第一个log, 返回下一个log的下标给leader即可.

(重新思考, 已改进)

```
以下这些情况, L->S的消息:
args:{
	log[4]
	preTerm = 6
	preIndex= 3
}

index  1  2  3  4
L:     4  6  6  6
S:     4  5  5
S->L   {5,2,3}
L->S   {log[2+0~4]}

L:     4  6  6  6
S:     4  4  4
S->L   {4,1,3}
L->S   {log[1+1~4]}

L:     4  6  6  6
S:     4
S->L   {-1,-1,2}
L->S   {log[2~4]}

L:     4  6  6  6
S:     5
S->L   {-1,-1,2}
L->S   {log[2~4], preTerm=4, preIndex=1}
S->L   {5,1,-1}
L->S   {log[1~4]}
```

Morris给出了他的设计, 让follow额外返回的信息:

1. XTerm: 如果一致性检验没有成功, 要么记录S的任期号, 要么设置-1表示S的`log[preIndex]`没有日志
2. XIndex: 如果XTerm不等于-1, 那么这里就记录下该任期在S的日志中的第一个log的index
3. XLen: 如果XTerm等于-1, 那么XLen记录的是S的日志长度(视频中第三个例子返回的是2不知道为什么)

所以以上三种情况的有效返回值是: `{5,2,-1}`; `{4,1,-1}`; `{-1,-1,2}`;

对于第一个返回值, leader得知冲突的日志任期是5, leader中找不到该任期的日志, 所以会覆盖其, 把nextIndex设置为2, 然后发送`log[2~4]`给S

对于第二个返回值, leader得知冲突的日志任期是4, leader中能够找到该日志, 所以会跳过其, 把nextIndex设置为1+1, 然后发送`log[2~4]`给S

对于第三个返回值, 思考了一下明白了, 可以看我自己写的第四个case, S的日志太短了, 所以直接追加日志即可, 发送`log[2~4]`

在第三个case的基础上, 我们再修改S的第一个日志任期为5, 这会被发送`log[2~4]`时的一致性检查发现, 然后返回`{5,1,-1}`给leader, leader发现自己没有该日志, 就直接从`log[1]`开始覆盖S的日志

**Q3:**

问题等价于: `raft如何保证leader拥有所有committed log entries`

论文中强调了*raft*的*rpcs*只会由*leader*发出(非选举), 所以我们要回答的是在这种情况下如何保持上述的性质的, 这一点在*raft*中通过选举制度来保证的, 以下是我的思考, 主要利用的性质还是两次**majority**必有重叠

```
1. 由于需要大多数followers写入日志才能提交, 所有对某一个committed log来说, 持有它的服务器数量一定大于N/2

2. 当选leader需要获得大多数的票数, 且仅持有更新的日志才会得票, 所以当选leader比超过N/2的服务器更新

3. 由于两次majority必定有重叠部分, 所以当选的leader必定持有该committed log

4. 归纳可得对每个committed log, 该当选leader必定持有
```


### Lab3B

先写一个大体框架吧

```go
Leader:
func (rf *Raft) Start(command) (...) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	rf.addLog(entries) // 写日志会修改一些东西, 稍后再谈
	rf.sendAppendEntries() // 并行通知其他服务器写日志
	
	return ...
}

Leader:
func (rf *Raft) sendAppendEntries() {
	args := AppendEntriesArgs{...}
	ApdLogCnt := 0
	for i := range rf.peers {
		if i != rf.me{
			go func(i int){
				reply := AppendEntriesReply{}
				rf.peers[i].Call("Raft.AppendEntries", &args, &reply)
				if reply...{...} // 返回值检查
				if ApdLogCnt > len(rf.peers) / 2{
					rf.commit() // 提交日志
				}
			}(i)
		}
	}
}

Follower:
func (rf *Raft) AppendEntries(args, reply){
	rf.mu.Lock()
	defer rf.mu.Unlock()

	rf.checkLog(args, reply) // 检查是否可以写入日志
	if reply.Success{
		rf.addLog(command) // 写入args中记录的日志条目
	}
}
```

先实现我们第一个函数`addLog(entries []Log)`, 当我们追加一个日志的时候, 哪些状态需要被修改?? 最基本的`rf.Log[]`肯定会追加内容, 然后对于leader的服务器, 我们有必要在实现的时候看看是否需要修改`nextIndex[]`的内容.

## Proof the Leader Completeness Property

**time:2024/5/19 23:22**

先罗列一下*Figure3*给出的五条*Raft*提供的保证, 并附带我自己的理解(原内容请直接看论文)

1. **Election Safety:** 每轮任期至多选出一位*leader*
2. **Leader Append-Only:** *leader*只会追加自己的日志内容, 从不覆盖
3. **Log Matching:** 这其实是两条性质, 对检查日志的一致性提供了很好的支持
   - 两条日志内容`log[i]==log[j]`当且仅当`i==j&&term_i==term_j`; 
   - 如果`log[i]==log[j]`, 那么`log[0~i]==log[0~j]`
4. **Leader Completeness:** 任期数更高的*leader*一定拥有任期数更低的*leader*提交过的日志内容, 换句话说, 每一期的*leader*拥有所有已提交日志内容
5. **State Machine Safety:** 所有服务器都将以同一个顺序执行相同的日志条目在其状态机上

我们来做一个简单的证明, 以便更深刻的理解这些性质.

**性质1和性质2**, 这完全由我们代码实现的算法所保证, 包括*leader*的选举和日志副本的创建. 第一条性质我们采取了投票的方式决定某一任期的*leader*, 更深一步, 这由我们规定每一个服务器在每一任期只允许投一票所保证. 第二条性质则由规定*raft*中所有的日志内容只能从`leader->follower`保证, 在论文中我们也看到了, 当出现日志不一致的时候, *leader*强制*follower*更新为自己的日志.

**性质3**, 论文中有提到两个事实`给定任期t与下标i, leader只会创建一个log[i]`和`性质2`, 我们可以这样考虑, 这等价于映射`(term, index)->command`是一个单射, 由于`term`是严格单增的, 同时性质2保证了`index`也是严格单增的(至少是不减的), 所以单射不言而喻. 性质3的第二点内容则是由*AppendEntries Rpc*附带的一个一致性检查保证的, 我们由归纳得到它.

**性质4**, 论文是采用反证去证明的, 在回答**Q3**时候, 我直接验证了`每一期的leader拥有所有已提交日志内容`这个性质, 主要还是通过归纳得到的, 这里我们按照论文的思路去证明: 假设$L_{k}$在任期*k*内提交了一条日志内容 $log_{k}$ , 而从 $L_{k+t}$ 开始的leader不包含这条日志.

1. 首先在任期*k*的时候, 必定有大于$\frac{N}{2}$的服务器持有日志$log_{k}$, 在第*k+t*轮任期中, 投给了$L_{k+t}$的服务器数量必定也是大于$\frac{N}{2}$的, 由于我们保证了已提交的日志是不会丢失的, 所以这两次子集必定有交, 也就是存在$S$, 既持有这条日志, 同时也投票给了$L_{k+t}$
2. 这与投票的限制冲突, *S*会投给$L_{k+t}$, 必然有$L_{k+t}$的日志比*S*更新, 至少一样长,如果是一样长,那么根据性质3可得到整个日志内容都是一致,矛盾; 所以$L_{k+t}$的日志会比*S*更长更新, 

## 幂等rpc

**Raft RPCs are idempotent**, 对于故障的followers和candidates, 论文再次提到无限的重试rpcs.

但是我还是想知道, 因为心跳消息会附带日志, 是否不必等待某一条rpc?


## Raft Locking Advice

- Rule1 : 最好是修补`-race`运行后给出的数据竞争条件
- Rule2 : 对于共享数据进行修改时, 而其他程序可能中途查看数据, 则应该在这期间使用锁
- Rule3 : 在等待的时候要释放锁资源, 比如发送RPC并等待回复和调用`time.Sleep()`时

```go
Example:
  rf.mu.Lock()
  rf.currentTerm += 1
  rf.state = Candidate
  rf.mu.Unlock()
```

像上面的这段数据`currentTerm`和`state`, 其他地方访问这些数据的时候也应该要请求锁`rf.mu`

- Rule4 : 要小心在释放锁和重新获取锁之间的假设, 比如下面这个例子(实际上我实现的时候也遇到了)

```go
  rf.mu.Lock()
  rf.currentTerm += 1
  rf.state = Candidate
  for <each peer> {
    go func() {
      rf.mu.Lock()
      args.Term = rf.currentTerm
      rf.mu.Unlock()
      Call("Raft.RequestVote", &args, ...)
      // handle the reply...
    } ()
  }
  rf.mu.Unlock()
```

在这个例子中, 不同线程之间读到的`currentTerm`可能会不一致, 所以我们可以提前在持有锁的时候复制它, 然后在每个线程中使用该复制的资源



## TAs' advice

不管是助教给的建议,还是实验主页给的建议,都反复强调了*figure2*的重要性;

第一个误区是心跳消息, 不只是简单的重置选举定时器, 还要执行图2中提到的各种检查;

```
If an existing entry conflicts with a new one (same index but different terms), delete the existing entry and all that follow it.
```

思考一下,如果这里follower收到的是一条过时的rpc消息该如何处理?

### 4个主要的BUG

***livelocks***

 材料提到了一种情况,即当一个leader被选出来后, 很快就发生了另一次投票, 造成这个原因主要有以下:

1. 确保严格按照图2的说明重置选举定时器:
    - 收到**当前**leader的*AppendEntries & heartbeat* rpcs时, 一定强调是当前, 如果是过时的则不会重置
    - 开始选举的时候
    - 投票的时候
2. 如果你是候选人, 但是选举计时器触发了, 那你应该启动另一次选举
3. 确保遵循了图2中的*Rules for Servers*

比如你在当前任期已经投了票, 但是收到的*RequestVote*附带的term比你高, 那么你应该先退位, 并采用他们的任期, 从而重置你的投票记录*votedFor*, 然后再处理这条rpc, 这样就可以参与该任期的投票, 不至于错过
 
***incorrect or incomplete RPC handlers***

图2中容易被忽略掉的细节:

1. 在某个步骤如果你发现*reply false*那应该立即返回, 而不执行后续的步骤
2. 如果收到*AppendEntries*的*prevLogIndex*超过了日志的末尾, 你应该按照日志不匹配的情况回复, 即false
3. 即使领导人没有发送任何*entries*内容, 也应该执行*AppendEntries*的检查
4. 不要忘记*AppendEntries*中提到的第五个限制
5. 必须严格执行"最新日志"的检查

***failure to follow The Rules***

1. 你需要一个专门的*applier*来检测当*commitIndex > lastApplied*时应用日志.
2. 确保定期检查是否需要引用日志, 或者使用条件变量等更细粒度的检测机制
3. 如果领导人发出rpc后被拒绝, 而拒绝的原因不是日志不一致, 那么你应该立即下台, 不做任何额外的操作
4. 领导人不允许将*commitIndex*更新到前一个任期的某个地方, 你需要特别检查*log[N].term == currentTerm*
5. 

***term confusion***

## lab3c 持久化

1. **currentTerm**

改变服务器当前任期的场景: `发起选举时候: sendRequestVote()`, `任意rpc收到回复发现自己任期过期时`, `收到消息更新自己的任期时`