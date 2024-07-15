# Grove: a Separation-Logic Library for Verifying Distributed Systems

*Before paper*

***为什么要阅读这篇论文?***

这篇文章是关于`Formal Verification For Distributed Systems`领域非常新的工作, 主要还是为了在分布式系统的形式化验证方向涉足了解一下. 对此, `lec`的计划是

1. 介绍一个被验证过的*case study distributed system*---**GroveKV system**
2. 介绍形式化验证的动机和一些bg
3. *Grove*要如何解释*GroveKV sys*这样的代码
4. 反思*Grove*的成就和其局限

**要求重点关注论文中的**

- **1.Introduce**
- **2. Motivation case studies**
- **3. Evaluation**

---

## *GroveKV*

我并不想多去关注这篇论文的内容, 因为我对如**CSL并发分离逻辑**这样的都没有接触过, 只是大致知道有**形式化验证**这样的概念存在就足够了, 就当是中途休息一下, 让我们看看示范用的*GroveKV*的一些设计.

对于想要修改服务器配置的需求, *GroveKV*使用了一个单独的服务器: **configuration service**, 和我们在*Chain Replication*中看到的一样, 假设这是一个永不出错的服务器, 也就是建立在*Raft*, *paxos*, *zab*上的一个集群.

看看*GroveKV*其他模块:

1. **replication**
   1. 使用*goroutines*复制到其他服务器中,
   2. 容忍*n-1*台服务器故障(*primary/back-up*模型), 
   3. 进行复制操作的时候必须保证持有锁, 或者其他同步措施
   4. 使用`nextIndex`来决定该顺序
   5. 等待集群所有服务器回复该复制请求才会回复客户端(*primary/back-up*模型)
2. **reconfiguration**
   1. 前面也提到了, 使用*configuration service*
   2. 将当前服务器组的信息`seal`, 也就是保存下来, 然后从中获取配置信息, 将其复制到新的服务器中
   3. 使用`epoch numbers`来保证只有一次*reconfiguration*成功, 类似于*raft*中使用的`Term`
   4. 只需要使用服务器组中的一个服务器配置信息就可以了(*primary/back-up*模型)
3. **reads based on leases**
   1. 因为主服务器可能并不知道某些配置变更, 这个问题对于*raft*而言就是领导人需要复制*Get*请求, 否则有读到的过时状态的风险
   2. 使用租约机制, 服务器可以在不和其他服务器协调的情况下处理*Get*请求, 不过这需要确保集群中时间是同步的, 可以使用*NTP*网络时间协议, 或者*AWS*的时间同步服务
   3. 每个租约过期前, 其他服务器不能进入新的*epoch*, 这确保了所有读操作都读到正确的数据


