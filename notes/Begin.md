# Begin

## 6.828 与 6.5840的差异

```c
LEC     paper6.824         paper6.5840
1       MapReduce          MapReduce
2       Go tutorial        Go tutorial
3       GFS                None
4       FTVM               Linearizability(NEW)
5       TGMM               The Go..(NEW)
6       Raft 1~5           GFS
7       Raft 7~end         Raft 1~5
8       ZooKeeper          Raft 7~end
9       CRAQ               ZooKeeper
10      Aurora             None
11      Frangipani         Grove(NEW)
12      6.033-9            6.033-9
13      Spanner            Spanner
14      FaRM               Chardonnay(NEW)
15      Spark              FaRM
16      M-atFacebook       DynamoDB(NEW)
17      COPS               Ray(NEW)
18      CT                 M-atFacebook
19      Bitcoin            OndemandCL(NEW)
20      BlockStack         Boki(NEW)
21      AnalogicFS         SUNDR(NEW)
                     22    Practical BFT(NEW)
                     23    Bitcoin
                     24    EWP(NEW)
                     25    None(NEW)
```

6.5840的实验多了一个, 应该是过渡的.

我决定选择```6.5840 2024Spring```这一版的来学习, 有相同的论文读完后选择看6.828的视频, 6.5840的没有视频或者说到时候不习惯看油管的翻译则自己多花时间反复读论文.

6.s081那边还差最后一个Network的lab, 以及剩下三篇论文, 这里为了打起精神想到先读一篇分布式领域的论文来感受一下.

### 第一篇论文MapReduce: Simplified Data Processing on Large Clusters

Starting time : 2024/4/23 21:00

```
2024/4/24

free seg:
                                       sleep or course           dinner 
          9:30--------11:00±0:30  ........................16:30..........17:30---------23:30
        maybe 10                     |                                          |        |
            |                        |                                          |        |
free tot:   |   1~2h                 |        0                   0             |   6h   |
            |                        |                                          |        |
TIM:        +-------------------------                                          +--------+
                  5x(avg: 6min)                                              4x(avg: 13min)
```


## 开始你的lab

### lab guidance

1. 关于go的一些资料: [Online Go tutorial](http://tour.golang.org/), [Effective Go](https://golang.org/doc/effective_go.html), 访问 [Editors](https://golang.org/doc/editors.html) 来设置go编译器.
2. 使用Go语言的竞态检测器[race detector](https://blog.golang.org/race-detector), 参数`go test -race`, 修复关于races的报告.
3. 关于`Raft`的建议: [guide of Raft](https://thesquareplanet.com/blog/students-guide-to-raft/).
4. 关于实验中的锁的建议: [Locking](https://pdos.csail.mit.edu/6.824/labs/raft-locking.txt).
5. 不知道这是什么: [structuring](https://pdos.csail.mit.edu/6.824/labs/raft-structure.txt).
6. 帮助理解系统中不同部分的`code flow`: [Diagram of Raft interactions](https://pdos.csail.mit.edu/6.824/notes/raft_diagram.pdf)
7. 勤用`print`语句, 你可以通过`go test > out`把输出结果定向到文件`out`中, 研究你插入的`print`的结果, 有助于解决bug.
8. 注意你调试信息的格式, 以便使用`grep`命令来搜索你想要的内容
9. `DPrintf`可能比直接调用`log.Printf`更有用, 访问[Go format strings](https://golang.org/pkg/fmt/)来学习Go的格式化输出.
10. 关于解析log output的建议, 这里暂时还不懂
11. 学习git的地方, 看起来需要系统认识一下git:[Pro Git book](https://git-scm.com/book/en/v2), [git user's manual](http://www.kernel.org/pub/software/scm/git/docs/user-manual.html)

### Advice on debugging

## LAB1 MapReduce

### Introduce

```
本次实验中, 你会构建一个简单的MapReduce系统;

你将实现以下内容:

一个worker进程, 用于执行程序Map和Reduce, 并且处理读写文件操作;

一个coordinator进程, 分派tasks给workers以及处理故障的workers

ps: lab中使用coordinator这个词, 而不是论文中的master
```

### Started

```
We supply you with a simple sequential mapreduce implementation in src/main/mrsequential.go. It runs the maps and reduces one at a time, in a single process. We also provide you with a couple of MapReduce applications: word-count in mrapps/wc.go, and a text indexer in mrapps/indexer.go. You can run word count sequentially as follows:
```

文件`src/main/mrsequential.go`是我们给你提供的一个简单的Sequential MapReduce实现, 这意味着它一次只运行一个map任务或reduce任务(这与分布式环境中的运行多个任务相对).

此外我们还提供给你了一些MapReduce的程序:

- `mrapps/wc.go`: 单词计数
- `mrapps/indexer.go`: 文本索引器: 貌似是记录下每个单词出现的位置, 比如Hello, world! Hello, everyone! 就会为Hello这个单词记录下[1, 3]的下标吗吧啊, 表示出现在第1和第3个单词的位置

程序`mrsequential.go`的输出在文件`mr-out-0`中, 其输入文件格式如同`pg-xxx.txt`, 你可以随意使用`mrsequential.go`中的代码, 你应该浏览一下`mrapps/wc.go`中的代码看看使用MapReduce的程序是什么样的.

### Job

让我们正式介绍一下本次实验你的任务.

你将实现一个分布式MapReduce, 由两个程序组成:

1. the coordinator
2. the worker

实际系统中, `workers`进程会运行在多台不同的机器上, 但是本实验你只有一台机器, 所以将在一台机器上运行这些任务. `workers`通过**RPC**与`coordinator`通信, 每个`worker`进程都会循环向`coordinator`请求`task`: 论文中提到的map或者reduce;

`coordinator`会将那些没有在合理的时间内完成任务(本实验设置该时间为10s)的`worker`的任务交给另外一个`worker`

我们提供了一些代码在`main/mrcoordinator.go`和`main/mrworker.go`(worker的"主要"例程)中, 你应该在`mr/coordinator.go`,`mr/worker.go`和`mr/rpc.go`中实现它们.

***在word-count上运行你的代码***

```sh
// 确保word-count是全新的
$ go build -buildmode=plugin ../mrapps/wc.go

// 当前目录: main
$ rm mr-out*
$ go run mrcoordinator.go pg-*.txt

// 在另一个窗口中
$ go run mrworker.go wc.so

// 当MapReduce的Job完成后应该显示的
$ cat mr-out-* | sort | more
A 509
ABOUT 2
ACT 8
...
```

我们提供给你了一个测试脚本: `main/test-mr.sh`

它会检查`wc`和`indexer`的MapReduce程序在输入`pg-xxx.txt`文件时是否产生正确的输出, 并且还会检查你的代码是否并行运行map和reduce的tasks, 以及是否能从运行任务时崩溃的worker中恢复.

如果你现在运行脚本, 它就会挂起, 因为coordinator永远不会结束:

```sh
$ cd ~/6.5840/src/main
$ bash test-mr.sh
*** Starting wc test.
```

你可以将`mr/coordinator.go`中的函数`Done()`中的`ret := false`修改为`ret := true`, 这样coordinator就会立刻退出, 如下:

```sh
$ bash test-mr.sh
*** Starting wc test.
sort: No such file or directory
cmp: EOF on mr-wc-all
--- wc output is not the same as mr-correct-wc.txt
--- wc test: FAIL
$
```

你的每一个reduce输出的文件名格式是`mr-out-X`, 测试完成后, 输出如下:

```sh
$ bash test-mr.sh
*** Starting wc test.
--- wc test: PASS
*** Starting indexer test.
--- indexer test: PASS
*** Starting map parallelism test.
--- map parallelism test: PASS
*** Starting reduce parallelism test.
--- reduce parallelism test: PASS
*** Starting job count test.
--- job count test: PASS
*** Starting early exit test.
--- early exit test: PASS
*** Starting crash test.
--- crash test: PASS
*** PASSED ALL TESTS
$
```

你可能会看到如下的错误, 忽略它

```sh
2019/12/16 13:27:09 rpc.Register: method "Done" has 1 input parameters; needs exactly three
```


### 环境配置

暂时按照这个流程走:[go环境配置](https://zhuanlan.zhihu.com/p/625866338)

