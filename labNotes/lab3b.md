# 3B

大概完成于5月20日~5月26日

# Log replicated

*raft*将**consensus problem 共识问题**划分为了三个子问题: 

- *Leader election*
- *Log replication*
- *Safety*

3a部分的实验已经让我们一览了领导人选举部分, 现在让我们来看看关于*raft*的日志备份, 以及如何提供高可用性的保障.

## Preview

在我的设计中, 用于维持领导人身份的心跳消息和为落后的服务器追加日志的rpc并没有区分开来, *leader*只是将需要追加的日志附加在`HeartBeat RPC`, 然后接收方简单的检查下其附加的日志项是否为空来决定这是一个简单的心跳消息还是追加日志的消息, 因此并没有单独的`AppendEntries RPC`.

- **涉及的RPCs(论文中)**: *Heartbeat RPC*(原文*AppendEntries RPC*)
- **相关的函数**
  1. `func (rf *Raft) HeartBeatGoroutine(heartTime int) {}`
  2. `func (rf *Raft) HeartBeatLauncher() {}`
  3. `func (rf *Raft) SendHeartbeat(to int, args AppendEntriesArgs) {}`
  4. `func (rf *Raft) HeartbeatHandler(args *AppendEntriesArgs, reply *AppendEntriesReply) {}`
  5. `func (rf *Raft) SendRPC(to int, rpc string, args interface{}, reply interface{}) bool {}`
  6. `func (rf *Raft) ApplyGoroutine() {}`
  7. `func (rf *Raft) Start(command interface{}) (int, int, bool) {}`
- **与上层的接口**: 因为到现在, 我们需要处理上层应用发出的一系列操作, 所以必须再次回顾一下与上层的接口, 最好我们需要审视一下[测试代码](../src/raft/test_test.go), 当然建立在完成了整个*lab3*的基础上, 否则不要提前获取关于测试的知识.
  1. 配置*raft*集群: `func make_config(t *testing.T, n int, unreliable bool, snapshot bool) *config {}`
  2. 检查集群对`Log[index]`的执行情况`func (cfg *config) nCommitted(index int) (int, interface{}) {}`
  3. 向集群发出请求`func (cfg *config) one(cmd interface{}, expectedServers int, retry bool) int {}`
  4. `func (cfg *config) begin(description string) {}`
  5. `func (cfg *config) end() {}`
  6. 控制`config.rafts[i]`连接/断开整个集群: `func (cfg *config) connect(i int) {}` / `func (cfg *config) disconnect(i int) {}`

---

## 客户端请求: 从发出到回复

1. 客户端的请求通过`rf.Start(cmd)`发给集群中的领导人
2. 领导人将该请求封装成日志的格式写入`rf.Log`
3. 运行的`HeartBeatGoroutine`线程观察到有新日志出现, 通过心跳消息发送给集群中其他服务器
4. 领导人通过心跳rpc的返回消息确认过半数服务器将该请求写入日志, 正式`commit`这条请求, 并且在下一轮心跳消息中通过`LeaderCommit`告知其他服务器
5. 每个服务器运行的`ApplyGoroutine`线程观察到`rf.CommitIndex > rf.LastApplied`, 通过`rf.ApplyCh <- msg`将该请求`apply`到上层状态机

## Proof Raft's guarantees

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


---

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


**Q3:**

问题等价于: `raft如何保证leader拥有所有committed log entries`

论文中强调了*raft*的*rpcs*只会由*leader*发出(非选举), 所以我们要回答的是在这种情况下如何保持上述的性质的, 这一点在*raft*中通过选举制度来保证的, 以下是我的思考, 主要利用的性质还是两次**majority**必有重叠

```
1. 由于需要大多数followers写入日志才能提交, 所有对某一个committed log来说, 持有它的服务器数量一定大于N/2

2. 当选leader需要获得大多数的票数, 且仅持有更新的日志才会得票, 所以当选leader比超过N/2的服务器更新

3. 由于两次majority必定有重叠部分, 所以当选的leader必定持有该committed log

4. 归纳可得对每个committed log, 该当选leader必定持有
```



## 幂等rpc

**Raft RPCs are idempotent**, 对于故障的followers和candidates, 论文再次提到无限的重试rpcs.

但是我还是想知道, 因为心跳消息会附带日志, 是否不必等待某一条rpc?