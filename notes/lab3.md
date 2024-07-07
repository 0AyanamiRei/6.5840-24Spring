# lab3 guide (new)

## 规定日志格式


1. 服务器之间的通信

按照`时间搓`+`通信类型`+`发起者->接收者`+`通信内容`的语义来打印日志

**请求投票**: `xxxxxx  [VOTE-1] p0(F,1)->p1 "Ask Vote"`, *p0*在任期1中向*p1*请求投票

**参与投票**: `xxxxxx  [VOTE-1] p1(F,0)->p0 "OKKKK"`, *p1*在任期1中投票给了*p0*

**拒绝投票**: `xxxxxx  [VOTE-1] p1(F,0)->p0 "Refuse Vote:(1,2,3...)"`, *p1*在任期1中拒绝投票给*p0*, 其中1,2,3表示拒绝的原因编号

**追加日志**: `xxxxxx  [APED] p0(L,2)->p1 "Aped Log(2~) {(xx,2), (xx,2)}"`, *p0*根据`nextIndex[1]`, 向*p1*追加了两条日志

**发送心跳**: `xxxxxx  [APED] p0(L,2)->p1 "Heartbeat"`

2. 打印事件的完成情况

按照`时间搓`+`事件类型`+`处理服务器`+`事件内容`的语义来打印日志, 这里尽可能多输出一些服务器的信息

**追加日志**: `xxxxxx  [APED] p1(L,2) "CMIT(3) APLY(2) Add Log[3~] {(xx,2), (xx,2)}"`, 追加了两条日志到*log[4]*和*log[5]*, 当前已提交日志下标是3, 已应用日志下标是2

**提交日志**: `xxxxxx  [CMIT] p0(F,2) "CMIT(3+2) APLY(2) Commit Log[3~] {(xx,2), (xx,2)}"`, 提交了两条日志*log[4]*和*log[5]*, 当前已提交日志下标是3+2, 已应用日志下标是2

**应用日志**: `xxxxxx  [APLY] p2(F,1) "CMIT(3) APLY(2+1) Apply Log[1] {(xx,1)}"`, 应用日志到*log[1]*, 当前已提交日志下标是3, 已应用日志下标是2+1

**开启选举**: `xxxxxx  [VOTE-1] p0(F,1) "Vote Start"`, *p0*开启了任期为1的本轮选举

**统计得票**: `xxxxxx  [VOTE-1] p0(F,1) "Got 3 votes"`, *p0*在任期1的选举中已获得3张票(这条消息只有得票过半才会开始发送)

**任期过期**: `xxxxxx  [HERT] p0(F,2) "p1(L,1) Out of data"`, 领导人*p1*的任期号过期

## Raft-选举

在我的*raft*实现方案中, 我记录了最近收到来自**当前任期的leader**rpc消息的时间, 根据这个时间, 以及每个服务器随机的*ms*, 可以判断
是否超时, 也就是, 我单独开了一个线程:**ElectionGoroutine**, 定期休眠*ms*, 然后检查距离上次收到rpc消息的时间.

如果超时, 那么调用*BeginElection()*, 并且严格按照论文图2中*Candidates*的要求执行, 唯一的不同是我没有利用*VotedFor*而是使用*VotedTerm*来追踪投票的记录;

考虑一下, 投票的RPC消息是否要重发, 或者说规定一个限制, 这里也算后续可以优化的选项, 我们可以根据选举超时的时间最小值来设置, 在该时间范围内都可以尝试重发丢失的RequestVote RPCs消息

当上leader后要记得发送一轮心跳消息

## Raft-追加日志

相比之前唯一的改进就是重新看了下图2的内容: `If AppendEntries fails because of log inconsistency: decrement nextIndex and retry`, 意思是如果是因为日志一致性检查而失败的心跳rpc, 那么应该**retry**

.... 我发现如果`SendHeartbeat`里的重发rpc的话, 那就会通不过

```go
for {
  if !rf.SendRPC(to, "HeartbeatHandler", &args, &reply) {
		return
	}

  if !reply.Success{
    ......
    continue
  }
}
```

大概是因为我的args在这里是固定的, 如果想要实现重发的话, 需要把args的构造放在这里面, 或者说就在这里修改一下args的内容

这里也是错误的, 浏览了一下日志, 如果在这里重发, 引发了一个问题, 日志不齐全的服务器当选了领导人, 具体原因我没有分析.

## 3c BUG

改了几天后, 现在已经有1998/2000的成功率, 分析了一个错误的日志, 发现是由于落后的服务器更新日志太慢导致的, 这里是部分内容:

```log
937088  [Info] p0(L,74) "CMIT(469) APLY(469) LogLen=493 log[last]={74 8727}"
937088  [Info] p2(F,74) "CMIT(141) APLY(141) LogLen=491 log[last]={64 7795}"

944659  [APED] p0(L,74)->p2 "Aped Log[491~492]"
949949  [APED] p0(L,74)->p2 "Aped Log[491~492]"
950701  [APED] p0(L,74)->p2 "Aped Log[490~492]"
952965  [APED] p0(L,74)->p2 "Aped Log[490~492]"
953723  [APED] p0(L,74)->p2 "Aped Log[415~492]"
955234  [APED] p0(L,74)->p2 "Aped Log[414~492]"
955993  [APED] p0(L,74)->p2 "Aped Log[414~492]"
// 客户端再次发送
971887  [APED] p0(L,74)->p2 "Aped Log[199~493]"
972645  [APED] p0(L,74)->p2 "Aped Log[199~493]"
973405  [APED] p0(L,74)->p2 "Aped Log[199~493]"
974159  [APED] p0(L,74)->p2 "Aped Log[198~493]"
974919  [APED] p0(L,74)->p2 "Aped Log[198~493]"
// 客户端再次发送
990817  [APED] p0(L,74)->p2 "Aped Log[161~494]"
991577  [APED] p0(L,74)->p2 "Aped Log[161~494]"
992366  [APED] p0(L,74)->p2 "Aped Log[158~494]"
993121  [APED] p0(L,74)->p2 "Aped Log[133~494]"
993877  [APED] p0(L,74)->p2 "Aped Log[158~494]"
994634  [APED] p0(L,74)->p2 "Aped Log[158~494]"
995439  [APED] p0(L,74)->p2 "Aped Log[158~494]"
996201  [APED] p0(L,74)->p2 "Aped Log[158~494]"
```

**解决方案**

1. 我能想到的问题根源是: `心跳消息`和`追加日志`绑定在一起, 同时实验又要求心跳消息不能太快, 如果日志的内容出现很多任期不同的冲突条目, 那么就需要非常多次心跳消息才能完整的追上领导人日志, 如果领导人能不断重发日志给这种服务器, 应该可以提高速度
2. 第二个问题是为什么会产生这么多不同任期的日志, 这可能又跟选举领导人的效率挂钩


## short review

时隔一个月, 再次回顾一下lab3c之前的代码框架

从3个方面来介绍: **领导人选举**, **日志追加**, **日志提交**

---

***领导人选举***

每个服务器设置一个线程用于发起投票, 每隔`350~550ms`进行一次检查, 由于我为每个服务器设置了一个`LastRPC`用于记录上一次收到rpc消息的时间, 所以检查该值与现在的时间差是否大于超时时间即可判断当前领导人是否故障.

若发起一轮投票:

发起者调用`BeginElection()`, 并行调用`SendElection`使用`SendRPC`发送rpc消息`RequestVote`给其他服务器请求投票

**日志追加**

客户端通过`Start()`将请求发给领导人, 领导人追加到自己的日志中, 发给其他服务器的日志随心跳消息一起;

心跳消息仅由领导人发出, 每隔`heartTime`就调用`HeartBeatLauncher()`发一轮心跳消息, 通过`NextIndex`和领导人自己的日志长度来决定是否需要附加日志发给其他服务器, 使用`SendRPC`发送rpc消息`SendHeartbeat`给其他服务器.

在对心跳消息的处理, 需要做一致性检查, 这里进行了优化, 对于一致性不通过的情况, 需要往回修改`NextIndex`, 如果一致性检查通过且领导人自身任期正常, 更新`MatchIndex`, 并且提交当前任期下大多数服务器持有的日志(顺带把以前任期, 但未提交的提交了), 这里一旦找到有需要提交的, 就通过`rf.ApplyCond.Broadcast()`唤醒*apply*线程, 应用那些可以应用的请求到状态机

**日志应用**

设置条件变量`ApplyCond`, 在每个服务器自身的`rf.mu`锁上等待, 只有当`rf.CommitIndex`有变动的时候才会唤醒这个线程

## 3D

第一件事我们需要知道何时创建快照, 需要考虑一个参数要么定时, 要么定量检测日志长度, 不过调用`Snapshot()`的任务暂时在这里交给了测试程序, 我们只需要实现就可以了.

当follower收到InstallSnapshot的rpc时, 使用用`ApplyMsg`将快照发给`applyCh`

``` go
type ApplyMsg struct {
	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}
```

发送快照的时候需要把`ApplyMsg.CommandValid`设置为*false*表示这不是一个指令

`Snapshot(index int, snapshot []byte)`, 第一个参数`index`表示快照蕴含的最高下标日志, *raft*应该删掉服务器中`index`之前的日志项.

***算上快照后的日志下标计算偏移***

现在的问题是, 在*raft*层的下标要怎么处理, 我们可在这层之间的计算都使用截断*Log*后的下标, 也就是虚假下标, 然后涉及到下标的比较的时候, 我们多传入参数`LastIndex`和`LastTerm`: 

*leader*传入$LastIndex_{leader}$, $LastTerm_{leader}$, 以及当前日志中的下标$index_{leader}$, 与*follower*进行比较, $LastIndex_{follower}$, $LastTerm_{follower}$, $index_{follower}$

1. **leader的这条日志比follower的新:**

$LastTerm_{leader}$ > $LastTerm_{follower}$;

$LastTerm_{leader}$ == $LastTerm_{follower}$ && $LastIndex_{leader} + Index_{leader}$ >= $LastIndex_{follower} + Index_{follower}$

2. **follower落后太多, 需要leader发送snapshot**

$NextIndex$ 

### snapshot后的index问题

先收集所有需要转换的下标:

- *Raft*层索引每个服务器的`rf.Log[index]`, `rf.Log[st:]`, `rf.Log[:ed]`
- 服务器维护的日志长度`rf.LogLength`
- *leader*为其他服务器维护的`rf.NextIndex[]`, `rf.MatchIndex[]`
- 已提交/应用的下标`rf.CommitIndex`, `rf.LastApplied`
- *RPCs*的参数, `RequestVoteArgs.LastLogIndex`, `AppendEntriesArgs.PrevLogIndex`, `AppendEntriesReply.XLen/XIndex`
- 与其他层的接口, `ApplyMsg.CommandIndex`, `Start`的返回值, `Snapshot`的参数

设计的时候, 还需要考虑到涉及到下标比较的时候要如何处理`什么时候leader需要发送snapshot`, `fast backup的一系列返回值内容, 以及一致性检查`

```
case1:
leader: lastIndex = 4,  log[0], log[1], log[2], log[3]
一个比较落后的follower:
	follower: lastIndex = 0, log[0], log[1]
一个对比的版本:
	follower: lastIndex = 4, log[0], log[1]
想一想, 怎么设计, followr返回什么信息才足够告诉leader需要发送snapshot
```

---

让我们开始解决这个问题, 定义**虚拟索引**为做过日志压缩后的索引, **全局索引**为不使用Snapshot时的索引.

先从解决实际问题开始比较好:

1. *raft*层的所有索引都处理为虚拟索引

leader当选后设置`NextIndex=len(rf.Log) =4`, 一致性检查的日志项为领导人中的`log[3]`.

- 那么第一轮心跳消息不附带日志, 领导人需要发送`LastIndex=4`, `PrevLogIndex=3`
- 显然第一轮一致性检查不会通过, (2a)被检查的日志条目不存在: `args.PrevLogIndex + args.LastIndex > rf.LogLength - 1 (7>1)`, *fast backup*优化中, 要求返回`XLen = rf.LogLength (2)`, 显然不足以和另一个对比版本的情况区分开, 返回`XLen = rf.LogLength + rf.LastIndex (2)`, 另一个版本则返回4+2=6, 显然, 我们可以根据领导人自己的`LastIndex`来区分, `XLen < LastIndex`则发*snapshot*
- 另一个对比版本, 领导人原有处理是`rf.NextIndex[to] = reply.XLen`, 因为这个方案下记录的都是虚拟地址, 所以减去`rf.LastIndex=4`即可, 下一次一致性检查就会通过.

显然`Snapshot`中的参数`index`是由上层`State Machine`传入的, 是全局索引.

`BeginElection()`中传入参数`args.LastLogIndex`由`rf.LogLength`传入, 所以是虚拟索引, 同时需要注意, `Log[0]`实际上保留的	`LastLog`, 所以当`rf.LogLength=1`的时候, 这里一致性检查的`LastLogTerm`是正确的

`SendElection()`中当选领导人后对`NextIndex`的初始化也是通过`rf.LogLength`, 故也是虚拟索引

`RequestVote()`里涉及到了新旧日志比较, 到这里我才意识到, 上面讨论的那个问题, 对于所有涉及日志比较的情况都需要讨论, 如果我们仅仅用虚拟索引去进行比较, 当某个服务器太过落后, 导致*snapshot*不同, 但是虚拟索引更大, 显得更新则会出现问题

```go
// leader: LastIndex=4, log[0], log[1] (term=4)

// follwer: LastIndex=0, log[0], log[1], log[2] (term=4)

//
args:= RequestVoteArgs{
		LastLogIndex = rf.LogLength - 1 (= 1)
		LastLogTerm = rf.Log[1].LogTerm (= 4) 
}

//3: 候选人日志不如自己新
args.LastLogTerm (=4) == rf.Log[rf.LogLength-1].LogTerm (=4)
args.LastLogIndex (=1) < rf.LogLength - 1 (=2)

// 可以看见这里比较结果是候选人更旧
```

上面的情况完全可能发生, 所以如果想要在*raft*层全部使用虚拟索引, 涉及到日期检查的rpc都需要领导人添加额外的信息, 至少需要`LastIndex`

---

不对, 发的时候就用全局索引就可以了, 然后比较的时候`rf.LogLength-1+rf.LastIncludedIndex`即可

`HeartBeatLauncher()`中准备发心跳消息, `PrevLogIndex`是用的虚拟索引(`PrevLogIndex=PrevLogIndex[i] = rf.NextIndex[i] - 1`), 检查`HeartbeatHandler()`发现, 可以传全局索引`PrevLogIndex`的

那么在`HeartbeatHandler()`中, 涉及到`PrevLogIndex`的比较, 都按全局索引比较即可, 这里涉及到*fast backup*, 所以详细解释代码:


```go
// 2a: 被检查日志条目不存在
	//if条件按全局索引比较
args.PrevLogIndex > rf.LogLength - 1 + rf.LastIncludedIndex
	//reply.XLen应该返回全局索引: 
reply.XLen = rf.LogLength + rf.LastIncludedIndex


// 2b: 被检查日志条目任期冲突
	//用leader的PrevLogIndex减去follower的LastIncludedIndex没有问题, 全局索引看来都是检查的同一条日志项
args.PrevLogTerm != rf.Log[args.PrevLogIndex - rf.LastIncludedIndex].LogTerm

// 删除包括该冲突日志在内的后续所有日志
rf.Log = rf.Log[:args.PrevLogIndex - rf.LastIncludedIndex]

// 附加了日志(这里通过了一致性检查+任期检查)
index := globalOffset + 1
rf.Log = append(rf.Log[:index], args.Entries...)

// AppendEntries RPC的第五条建议 需要唤醒apply线程
// rf.CommitIndex还是设置为全局索引比较好
rf.CommitIndex = Min(args.LeaderCommit, rf.LogLength-1+rf.LastIncludedIndex)
```

ok,总结一下, 用于*fast backup*的三个参数都是全局索引, 记录已提交/应用的是全局索引.

看如何通过这些参数在领导人这边修改数据:

```go
// 一致性检查+任期检查通过, NextIndex初始化的时候设置的是虚拟索引, 而rpc的参数PrevLogIndex是全局索引
rf.MatchIndex[to] = args.PrevLogIndex + len(args.Entries) - rf.LastIncludedIndex

rf.NextIndex[to] = rf.MatchIndex[to] + 1 
// 所以NextIndex, MatchIndex是虚拟索引, 但是记得CommitIndex是全局

// 一致性检查不通过
// rf.Log本身就是虚拟的, 所以任期冲突的代码不修改

// 尽管XLen的值涉及到不同服务器的LastIncludedIndex, 但全局的含义是一样的
rf.NextIndex[to] = reply.XLen - rf.LastIncludedIndex
```

`Start()`中, 返回值是提供给上层函数, 所以必须是全局索引, `return rf.LogLength - 1 + rf.LastIncludedIndex`

`ApplyGoroutine()`中, `CommitIndex`和`LastApplied`都当全局索引处理, 所以中途需要转换一下

*ok*, 测试一下3B/3C: `python dtest.py 3B 3C TestSnapshotBasic3D -n 5 -v -p 10`
