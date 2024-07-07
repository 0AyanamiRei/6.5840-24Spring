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

- `Next`