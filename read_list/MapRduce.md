# MapReduce: Simplified Data Processing on Large Clusters

## 摘要和介绍

介绍了**MapReduce**是什么, 以及其主题内容的两个函数: ```map()```, ```reduce()```, 然后介绍了一下**run-time machine**的作用, 笔者猜测大致的架构是一个或多个```run-time machine```来进行摘要提到的管理的工作, 然后下面有一大批```commodity machines```运行着用户想要执行的任务.

为了快速处理大量的数据, 我们不得不将计算的任务分发到许多机器上, 这也产生以下这些问题, 让原本简单易懂的计算变得十分复杂

1. parallelize the computation
2. distribute the data
3. handle failures
4. load balancing
5. ...

为此作者团队设计了**MapReduce**这一抽象层, 对外表达的是简单易懂的运算, 在内部隐藏了该层实现时复杂的细节.

这里是这篇论文后续section 2-7的主要内容:

- 2: 基础的编程模型
- 3: MapReduce接口的实现
- 4: 一些改进
- 5: 在不同任务下的性能测试
- 6: 使用MapReduce重写了谷歌自己的indexing sys
- 7: 讨论了一些相关和未来的工作

## 建立MapReduce的心智模型

笔者将通过回答以下这些问题以构建一个关于**MapReduce**的心智模型, 但是请注意, 虽然不会一开始就陷入到很多细节中, 但是在对主体有了一个大致认识后,还需更多深入的思考.

1. ``MapReduce``对外的接口, 使用起来会是什么样的?
2. 库中的``Map``和``Reduce``是如何使用的, 二者之间是如何连接的?(其实我很想问为什么要采用论文中这样的设计模式,但是我觉得初学应该先"看")
3. ``MapReduce``的世界中, 当我们调用接口的时候, 运算集群内部的PC如何把任务分解的?

***1. MapReduce对外的接口, 使用起来会是什么样的?***

```c++
#include "mapreduce/mapreduce.h"
// 用户自定义Map...
// 用户自定义Reduce...
int main(int argc, char **argv)
{

  // 设置输入的文件
  MapReduceSpecification spec;
  for (int i = 1; i < argc; i++)
  {
    MapReduceInput *input = spec.add_input();
    input->set_format("text");
    input->set_filepattern(argv[i]);
    input->set_mapper_class("WordCounter");
  }

  // 设置输出的文件
  MapReduceOutput *out = spec.output();

  // 设置参数
  out->set_num_tasks(100);
  spec.set_machines(2000);
  spec.set_map_megabytes(100);
  spec.set_reduce_megabytes(100);

  // 执行任务
  MapReduceResult result;
  MapReduce(spec, &result)

  return 0;
}
```

使用**MapReduce**的程序员只需要设置``Map``和``Reduce``函数, 并设置一些参数即可享用分布式运算带来的高性能, 这些事情你只需要去阅读一些使用该库的手册之类的即可完成, 完全不需要拥有其内部实现的知识.

***2. 库中的Map和Reduce是如何使用的, 二者之间是如何连接的?***

文章中的使用该库的问题假设以一系列``key/value pairs``作为输入,然后以一系列``key/value pairs``作为输出, ``MapReduce``将这个过程分解成了``Map``和``Reduce``.

1. **Map过程**: ``Map``接收一系列的输入, 输出``intermediate key/value pairs``(中间键值对), ``MapReduce``库会自动地把``Map``产生的这些中间键值对**捆绑**: <u>把具有相同key的key/value pairs关联到一起, 以集合I表示这些value</u>
2. **Reduce过程**: ``Reduce``函数通过key/I接收``Map``函数产生的中间键值对, 对某一个key对应的value集合I做一些处理, 比如如果值是数量类型的属性, 就可以简单的做一些相加的工作.

***3. MapReduce的世界中, 当我们调用接口的时候, 运算集群内部的PC如何把任务分解的?***

要回答这个问题, 我们可以直接拿出论文中的Figure1, 十分的清晰, 不过在这之前, 让我们做一些定义:

- **Job**: 这是用户视角下的, 指的是交给```MapReduce```的一次任务
- **task**: 文章中有两个task, map/reduce task, 交给MapReduce的Job会被分解成许多这种小规模的tasks, 每个task的状态有```idle```,```in-progress```, ```completed```.
- **Master**: 即上文的```run-time machine```, 管理tasks,记录它们的状态, 同时也负责调度机器来完成这些tasks, 文章中仅有一台这样的机器.
- **Worker**: 执行被分配的tasks, 通常规模庞大, 论文当时是以2k台机器的规模来进行讨论.

<img src="./MapReduce架构.png" width="500" height="300">

图示最左和最右侧是输入的文件和输出的文件, 这些文件是放在一个全局的文件系统中, 当时谷歌使用的是GFS: 一种分布式的文件系统, 大致可以理解为自动将一个大型文件分解成若干64MB大小的blocks进行存储, 这些```split 0~4```便是一个一个小的blocks.

执行map task的机器将中间键值对存储在其本地的磁盘中, 这是因为当时谷歌使用MapReduce的性能瓶颈是网速, 所以他们尽可能设计地来避免使用网络, 或者说不让整个Job的时间都花费在等待网络传输数据, 所以map tasks产生的中间键值对并没有实时地传递给执行reduce task的机器. 

## Fault Tolerance
