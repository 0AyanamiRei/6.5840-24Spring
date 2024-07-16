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


so, 当前的设置

- 全局索引: `Snapshot(index int)`,  `args.PrevLogIndex`, `reply.XLen`, `rf.CommitIndex`, `reply.XIndex`, `Start()返回值`, `rf.LastApplied`, `args.LeaderCommit`
- 虚拟索引: `args.LastLogIndex`, `rf.NextIndex[to]`, `rf.MatchIndex[to]`

### 实现follower的InstallSnapshot函数

1. 常规的RPC检测, 更新自己的任期
2. 拒绝任期更小的leader的一切rpc消息
3. 根据全局索引来比较, 如果自己的`Snapshot + rf.Log`覆盖了该rpc附带的`snapshot`, 那么就直接`return`
4. rcp附带的`snapshot`更全, 那么截取所有其包含的日志信息
5. 修改`CommitIndex`和`LastApplied`
6. 应用`snapshot`

### leader需要发送Snapshot的场景

只要是涉及到leader给其他服务器追加日志的时候,就可能需要调用`SendSnapshot()`, 比如

1. `reply.XLen - rf.LastIncludedIndex < 0`, 这种情况, follower的日志完全被领导人的snapshot覆盖, 同时修改`NextIndex[to]=1`
2. `reply.XIndex - rf.LastIncludedIndex < 0`, 这种情况, follower多余的日志都是无用的, 等于被snapshot覆盖


`python dtest.py TestSnapshotBasic3D TestSnapshotInstall3D -n 20 -v -p 10`

`python dtest.py TestSnapshotBasic3D TestSnapshotInstall3D TestSnapshotInstallUnreliable3D TestSnapshotInstallCrash3D TestSnapshotInstallUnCrash3D -n 20 -v -p 10`

**完整测试** `python dtest.py 3A 3B 3C 3D -n 20 -v -p 10`

---

### 规范索引

初次写完3D的代码后测试的结果很不理想, 代码逻辑也非常混乱, 故这里统一规定

1. 借用操作系统的命令方式, 将全局的下标索引记为**虚拟索引**, 然后把`rf.Log[index]`实际使用的记为**物理索引**, 调用`V2PIndex(idx)`和`P2VIndex(idx)`来互相转换. 
2. 同时规定, 关于索引, 记录的内容全部采取**虚拟索引**, 因为第一次写完我发现涉及到**下标比较**的时候, 都需要或者说使用全局的下标比较方便, 不过`rf.LogLenth`算一个例外, 因为直接写`=len(rf.Log)`比较简单, 不用再加一个值
3. `LastIncludedTerm`使用`rf.Log[0]`记录, 这个位置的日志项我们没有使用.
4. 日志信息中出现的所有下标直接展示的物理索引(不过列表的表示还是左闭右开), 然后紧跟一个括号表示虚拟索引, 表达日志的格式为`Log[1~a+b]`, 表示`LastIncludedIndex`为`a`, 然后整个日志的最后一个索引值是`a+b`(`LastIncludedIndex + LogLength`)

### InstallSnapshot

久经折磨, 终究还是一字一句的遵守论文中图13已经图8的指示:

1. `Reply immediately if term < currentTerm`, 常规的rpc处理程序检查
2. 更新自己的计时器
3. `If existing log entry has same index and term as snapshot's last included entry, retain log entries following it and reply`, 这句话的意思是从现有的日志中找到这样一个匹配的日志项, 其实就是确认快照的最后一个日志项是否在自己的日志中能找到
4. 将快照应用到上层状态机, 并且修改自己的配置参数

### 发送快照的时机

简单来说, 我觉得只要涉及到领导人自身日志项的环节就都有可能出现, 待使用日志被快照覆盖, 从而导致无法在现有日志中找到. 

不过先想好什么时候要发快照, 当访问日志`rf.Log[index]`出错的时候, 即`index < 0`, 同时, 如果是要附加日志内容, 那么这里应该是`index <= 0`, 因为第零项没有实质内容

在我设计里, 以及通过前面的经验, 我应该关注下面两个过程;

1. `HeartBeatLauncher()` 心跳发射器为其余服务器的`args`发送rpc考虑是否需要附带日志项的时候
2. `SendHeartbeat()` 心跳发射函数, 收到`reply`后需要修改`NextIndex`, 这个时候如果我们发现需要退回的日志已经在快照中, 就可以直接发快照

### 其他内容

还需要进行修改的就是当一致性检查的日志是`PrevLogIndex=0`时, 因为暂时不确定`rf.Log[0]`的任期就是快照的最后一个日志项任期, 所以还是用`rf.LastIncludedTerm`来进行检查

### BUG记录

1. 报错的信息是`panic: runtime error: index out of range [-8]`, 出错的代码:`PrevLogTerm[i] = rf.Log[rf.V2PIndex(rf.NextIndex[i])-1].LogTerm`, 此时的`NextIndex: = [14 1 2]`, 分析结果是`p2`因为断连不跟领导人联系, 那么`NextIndex[2]`就一直保持了断开网络前的状态, 也就是2, 但此时领导人的`LastIncludedIndex`是9, 所以导致了这里索引为负值, 主要还是`NextIndex`不更新的原因, **为什么不检查出来就直接发snapshot呢?**
2. 崩溃后重启的服务器会从快照和持久化数据中恢复数据, 由于我们并没有持久化`CommitIndex`和`LastApplied`, 所以恢复后的服务器只有快照加保存的日志内容, 这个时候当前的领导人会追加日志到该服务器, 然后在`HeartbeatHandler()`更新自己的`CommitIndex`, 但是因为`LastApplied=0`, 明显`rf.CommitIndex > rf.LastApplied`, 所以应用日志的时候会导致`rf.V2PIndex(rf.LastApplied) < 0`, 出现负数索引, 所以要么我们持久化该信息, 要么在修改`CommitIndex`的同时保证`LastApplied`不小于`rf.LastIncludedIndex`
3. `TestSnapshotAllCrash3D`中, 如果所有机器都挂掉, 然后重启, 那么会出现`2`中提到的问题, 因为这个时候