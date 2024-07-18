本仓库是我的MIT6.5840(原MIT6.824)课程日志追踪, 将自己过程中所见所识记录于此.

## Read list

除了今年课程要求的阅读材料, 我也选取了往年6.824要求的阅读材料(差异还是比较大, 但是主体部分不会相差很多)

- [x] [MapReduce](https://pdos.csail.mit.edu/6.824/papers/mapreduce.pdf): 分布式计算的一个设计方案, 或者说是模式...?
- [x] [GFS](https://pdos.csail.mit.edu/6.824/papers/gfs.pdf): 分布式的文件系统, 早期版本
- [ ] [Fault-Tolerant Virtual Machines](http://nil.csail.mit.edu/6.824/2020/papers/vm-ft.pdf): 应该是主从模型
- [x] [Testing Distributed Systems for Linearizability](https://www.anishathalye.com/2017/06/04/testing-distributed-systems-for-linearizability/): 读完后会对分布式中的**一致性**有一个大概的认识
- [x] [Raft(Extended)](https://pdos.csail.mit.edu/6.824/papers/raft-extended.pdf): 非常经典的**共识**算法, 也有大量的优化在原作者的博士论文中
- [x] [ZooKeeper](https://pdos.csail.mit.edu/6.824/papers/zookeeper.pdf): 使用**ZAB**(类似*raft*)构建一个以**znode**, 对象为模型的存储系统, 利用其提供的一些接口, 我们能构建出很实际的需求
- [x] [Chain Replication](https://www.cs.cornell.edu/home/rvr/papers/OSDI04.pdf): **链式复制模型**, 在进行`lab3`后, 对*raft*有了一定认知后, 会对这种非**主从**的复制模型感到惊奇, 同样是强一致性模型, 但看起来要简单很多, 且在读多写少的场景下表现很好
- [x] [CRAQ](http://nil.csail.mit.edu/6.824/2020/papers/craq.pdf): **链式复制模型**的改良, 阅读起来有一定难度, 只看明白了**Introduction**中提到的文章贡献第一二点.
- [ ] [Grove](https://pdos.csail.mit.edu/6.824/papers/grove.pdf): 分布式系统的形式化验证, 指引我们设计出一个能对所有极端情况做出正确反映的分布式系统, 主要内容没有看, 暂时没有这个能力.
- [ ] [Spanner](https://pdos.csail.mit.edu/6.824/papers/spanner.pdf)
- [ ] [Chardonnay](https://pdos.csail.mit.edu/6.824/papers/osdi23-eldeeb.pdf)
- [ ] [FaRM](https://pdos.csail.mit.edu/6.824/papers/farm-2015.pdf)
- [ ] [DynamoDB](https://pdos.csail.mit.edu/6.824/papers/atc22-dynamodb.pdf)
- [ ] [Ray](https://pdos.csail.mit.edu/6.824/papers/ray.pdf)
- [ ] [Scaling Memcache at Facebook](https://pdos.csail.mit.edu/6.824/papers/memcache-fb.pdf)
- [ ] [On-demand Container Loading in AWS Lambda](https://pdos.csail.mit.edu/6.824/papers/atc23-brooker.pdf)
- [ ] [Boki: Stateful Serverless ComputingwithSharedLogs](https://pdos.csail.mit.edu/6.824/papers/jia21sosp-boki.pdf)
- [ ] [Secure Untrusted Data Repository](https://pdos.csail.mit.edu/6.824/papers/li-sundr.pdf)
- [ ] [Practical Byzantine Fault Tolerance](https://pdos.csail.mit.edu/6.824/papers/castro-practicalbft.pdf)
- [ ] [Bitcoin: A Peer-to-Peer Electronic Cash System](https://pdos.csail.mit.edu/6.824/papers/bitcoin.pdf)


## Lab

如果您也在进行该课程附带的实验内容, 那么我不建议您翻阅我的源码, 一来是为了锻炼自己的能力, 遵守学术诚信, 二来是我自知我的代码难以阅读, 设计上会给您带来极大的困扰

- [x] Lab1: MapReduce [实验日志](./labNotes/lab1.md)
- [ ] Lab1: Challenge 
- [x] Lab2: Key/Value Server [实验日志](./labNotes/lab2.md)
- [x] Lab3A: (raft)Leader election [实验日志](./labNotes/lab3a.md)
- [x] Lab3B: (raft)Log [实验日志](./labNotes/lab3b.md)
- [x] Lab3C: (raft)Persistence [实验日志](./labNotes/lab3c.md)
- [x] Lab3D: (raft)Log Compaction ( Snapshot ) [实验日志](./labNotes/lab3d.md)
- [ ] Lab4A: Key/value service without snapshots
- [ ] Lab4B: Key/value service with snapshots
- [ ] Lab5A: The Controller 和 Static Sharding
- [ ] Lab5B: Shard Movement

## Not Just Lab

完成上述`Lab`的过程中, 我检查了大量的错误日志, 也从阅读材料和其他人的设计中学到了许多, 思考过后打算对原有的实验内容进行改动, 自己去实现一遍, 在过程中锻炼自己.

[my raft](https://github.com/0AyanamiRei/6.5840-24Spring/blob/main/labNotes/my%20Raft.md)

**可能有帮助的阅读材料**

1. [分布式系统中的网络模型和故障模型](https://danielw.cn/network-failure-models)
2. [Symmetric network partitioning](https://github.com/baidu/braft/blob/master/docs/cn/raft_protocol.md#symmetric-network-partitioning)
3. [一个极端的网络故障场景](https://www.zhihu.com/question/483967518),[CheckQuorum](https://github.com/etcd-io/etcd/issues/3866)
4. 开源的一些实现, 或许能从中学到更多测试, 以及优化实现方案: [braft(C++)](https://github.com/baidu/braft) [jraft(Java)](https://github.com/sofastack/sofa-jraft/tree/master)

- [ ] 更完善的测评: 实验自带的测评并没有cover所有情况, 甚至可以说只保证了大部分情况下能用, 但是很多极端的环境下都未进行测试
- [ ] 更完善的Raft
