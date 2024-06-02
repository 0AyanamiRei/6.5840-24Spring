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