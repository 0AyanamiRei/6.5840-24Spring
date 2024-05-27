# 这里是一些阅读材料的笔记

## Notes on Distributed Systems for Young Bloods

[Notes on Distributed Systems for Young Bloods](https://www.somethingsimilar.com/2013/01/14/notes-on-distributed-systems-for-young-bloods/)

```
The social stuff is usually the hardest part of any software developer’s job, and, perhaps, especially so with distributed systems development.
```

正文开始前,  很有意思的一个问题,  所谓`social stuff`也是很重要的.

第一段内容是关于**分布式系统中的故障**

作者指出, 一些人(年轻的工程师)可能会认为**延迟**是导致其极其困难的原因, 但是实际上是**故障**, 这并不是"well, it’ll just send the write to both machines"或"it’ll just keep retrying the write until it succeeds"能够简单解决的, 因为在分布式系统中所面临的故障, 经常是**部分性的故障**, 这意味着在一部分机器上成功, 另一部分却失败, 那么如何保持一致性, 这部分更为困难,

接下来的两个问题出自**分布式系统的现实问题**, 对的, 就是昂贵, 不管是编写一个强大的分布式系统还是长期维护一个开源的分布式系统, 都不是单个人所能承担的花费(所以很难看到一个优秀的开源分布式系统项目), 然而很多故障, 比如网络, 只有在多机的情况下才能观察到, 同时单机的数据集容纳量也是远不如多机的, 这些问题是即使在能够使用**虚拟机**和**云技术**来模拟的情况下也是存在的.