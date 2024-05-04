# 解读函数

## wc.go

`wc.go`文件定义了**MapReduce**程序的`Map`和`Reduce`函数:

1. `Map(filename, contents)`, 两个参数分别是`文件名`和`文件内容`, 然后将文件内容分割成单词, 每个单词生成一个键值对:`<word, 1>`

2. `Reduce(key, values[])`, 第一个参数key即为一个单词, 然后values列表在这里实际就是[1,1,1...], 其长度就代表了这个单词出现在文件中的次数.

我们来具体看看代码,主要是想了解下不同文件之间传递了哪些数据结构

```go
// worker.go中
type KeyValue struct {
	Key   string
	Value string
}

func Map(filename string, contents string) []mr.KeyValue {

	ff := func(r rune) bool { return !unicode.IsLetter(r) }

	words := strings.FieldsFunc(contents, ff)
	kva := []mr.KeyValue{}

	for _, w := range words {
		kv := mr.KeyValue{w, "1"}
		kva = append(kva, kv)
	}

	return kva
}
```

`mr.KeyValue`就是上面在文件`worker.go`中定义的结构体, `Map`函数把`contents`中的每一个单词处理成`<word, 1>`键值对存放在`kva`中,作为函数的返回值返回.

```go
func Reduce(key string, values []string) string {
	return strconv.Itoa(len(values))
}
```

统计一个单词, 也就是`key`在文本中出现的次数, 作为字符串形式返回其出现的次数.

## mrsequential.go

```go
if len(os.Args) < 3 {
  fmt.Fprintf(os.Stderr, "Usage: mrsequential xxx.so inputfiles...\n")
  os.Exit(1)
}
mapf, reducef := loadPlugin(os.Args[1])
```

`os.Args`, 这是存储`go run xxx传入的参数`的string数组, 示范代码中输入了:

```sh
go run mrsequential.go wc.so pg*.txt
```

<<<<<<< HEAD
其中`pg-*.txt`是一个shell通配符, 匹配当前目录下所有以`pg-`开头的文件.

## 开始实验

我们先搭起MapReduce的整体框架, 先完成仅启动一个worker和一个master的MapReduce, 为此, 先分两个部分来设计, **RPC**和**MetaData**部分

### MetaData

按照论文的要求, 以及我的实现的需求定义了如下的数据结构

```go
// 管理tasks的数据结构
type tasksMetadata struct {
	TaskType  int    // 任务类型 0:Map,1:Reduce
	TaskNum   int    // 任务编号
	TaskState int    // 任务状态 0:IDLE,1:INPROC,2:COMPLETED
	Filename  string // 读取文件名 
}

// Master的数据结构
type Coordinator struct {
	MapTasks    []tasksMetadata // 管理Map任务
	ReduceTasks []tasksMetadata // 管理Reduce任务
	NReduce     int             // 整个JobReduce任务数量
	NMapped     int             // 已完成的Map任务数量
	Nfile       int             // 整个Job读取的文件数量(Map任务数量)
}

//RPC使用的数据结构
type ExampleArgs struct {
	// worker->master
	Task tasksMetadata // 追踪任务信息
}

type ExampleReply struct {
	// master->worker
	Task    tasksMetadata // 追踪任务信息
	JobDone int           // JOBDONE:Job完成, JOBDOING:Job正在进行
}
```


### RPC

我们要提供足够的rpc通信函数

1. AskTasks: worker向主机请求分配任务
2. Task: 主机分配给worker任务
3. CommitTask: worker完成任务提交信息
4. CommitOK: 回应提交

第一对rpc通信是`AskTasks()`和`Task()`, 是由worker向主机发起的rpc, 由于是在一台电脑上模拟的, 所以我们要把worker所在进程的`PID`附加发给主机, 然后主机那边需要根据当前Job的执行进度来决定分配哪个task给这个worker, 同时对一些metadata进行修改, 所以我们的`AskTasks()`函数需要一个与任务有关的数据结构返回值

```go
// 我们返回一个记录了任务信息的数据结构
func AskTasks() tasksMetadata {
	args := ExampleArgs{
		Pid: os.Getegid(),
	}
	reply := ExampleReply{}
	if ok := call("Coordinator.Task", &args, &reply); !ok {
		log.Fatal("call to Coordinator.Task failed")
	}
	return reply.Task
}

// 为worker调度任务的同时修改记录的数据
func (c *Coordinator) Task(args *ExampleArgs, reply *ExampleReply) error {
	var task *tasksMetadata
	var num int
	Task := &c.MapTasks
	tasktype := MAP
	if c.NMapped == c.Nfile {
		Task = &c.ReduceTasks
		tasktype = REDUCE
	}

	for i, x := range *Task {
		if x.TaskState == IDLE {
			task = &(*Task)[i]
			num = i
			break
		}
	}

	// 修改task的元数据
	task.TaskType = tasktype
	task.TaskNum = num
	task.TaskState = INPROC

	reply.Task = *task

	// 调试信息
	if tasktype == REDUCE {
		fmt.Printf("(Master数据): 第%d号Reduce任务,任务状态:%d,对象文件名:%s\n",
			num, c.ReduceTasks[num].TaskState, c.ReduceTasks[num].Filename[0])
	} else {
		fmt.Printf("(Master数据): 第%d号Map任务,任务状态:%d, 执行PID:%d,对象文件名:%s\n",
			num, c.MapTasks[num].TaskState, c.MapTasks[num].Filename[0])
	}

	return nil
}
```

当worker完成任务后需要通知主机, 同时也要把任务完成的一些信息告诉主机.

```go
func CommitTask(task tasksMetadata, filename string) {
	args := ExampleArgs{
		Task:     task,
	}
	reply := ExampleReply{}

	if ok := call("Coordinator.CommitOK", &args, &reply); !ok {
		log.Fatal("call to Coordinator.CommitOK failed")
	}
}

// 处理结束的任务
func (c *Coordinator) CommitOK(args *ExampleArgs, reply *ExampleReply) error {
	num := args.Task.TaskNum
	Task := c.MapTasks

	if args.Task.TaskType == REDUCE {
		Task = c.ReduceTasks
		c.NReduce++
	} else {
		c.NMapped++
	}
	Task[num].TaskState = COMPLETED
	return nil
}
```

现在我们还没有去处理Map产生的中间文件, 也暂时不去管Map和Reduce的具体过程, 就当空语句跳过而已, 现在我们要完善整个框架让MapReduce能启动且正常结束, 为此还需要考虑一些事情.

我们的worker是处在一个循环中的, 不停的向主机请求任务, 我们设置当NReduce的值减少至0的时候(默认Map阶段完成, 因为只有Map完成才会开始Reduce)表示整个Job完成

我们添加一个Rpc: `Done`, mrcoordinator程序不断向主机查询是否JOB完成, 从而来退出循环关闭程序.

```go
func (c *Coordinator) Done() bool {
	if c.NMapped != c.Nfile || c.NReduce != 0 {
		return false
	}
	for _, task := range c.MapTasks {
		if task.TaskState != COMPLETED {
			return false
		}
	}
	for _, task := range c.ReduceTasks {
		if task.TaskState != COMPLETED {
			return false
		}
	}
	return true
}
```

这里我们暂时保守地对MapTasks和ReduceTasks记录的数据都检查了一遍, 其实只需要检查`NMapped==Nfile`和`NReduce==0`即可的. 添加一些输出语句, 来检查一下:(修改nReduce=2, 且只传入一个文件)

```sh
go build -buildmode=plugin ../mrapps/wc.go

go run mrcoordinator.go pg-grimm.txt pg-being_ernest.txt

go run mrworker.go wc.so
```

我们需要在`coordinator.go`中分配任务函数`Task`里处理`task == nil`的情况, 分析了一下原因, 我们主机进程每次调用`Done()`查询后就会休眠1秒, 这期间已经足够worker进程发出很多次询问了, 所以我们可以拦截`task == nil`返回一些信息给worker告诉worker进程Job已经完成了.

在`reply`里附加一个信息: `JobState` : RPCERROR = 2,JOBDONE = 1, JOBDOING = 0

现在再执行上面的语句后的结果:

```sh
(Rpc数据): 第0号MAP任务,任务状态:1,对象文件名:pg-grimm.txt
do....
over!
(Rpc数据): 第0号REDUCE任务,任务状态:1,对象文件名:CC
do....
over!
(Rpc数据): 第1号REDUCE任务,任务状态:1,对象文件名:CC
do....
over!
Worker关闭, JOB完成!
```

那么我觉得现在整体的框架就搭好了, 至少对目前简单版本的MapReduce

### Map()

先罗列一下Hints中对简单版本MapReduce完成task有帮助的:

1. 在实验的环境下, 我们允许利用workers共享同一个文件系统这一条件.
2. Map产生的临时文件一种易读的命令方式是`mr-X-Y`, 表示第X个Map任务, 第Y个Reduce任务.
3. 提示给出了一个临时文件中的建议: 使用go的包`encoding/json package`
4. 对于一个给定的key, 你可以使用`ihash(key)`来找到对应的reduce task编号
5. 你可以从`mrsequential.go`借鉴一些代码, 用于读取文件, 对中间键值对排序, 以及将Reduce的输出存储在文件

思考了一下, 现在可以理解为什么论文中说master要做O(MxR)次调度决定以及保存O(MxR)	个状态了, 同时我们要在master中管理map任务产生的中间文件, 我们在woker对master的RPC函数参数中附带这些信息: 产生的`mr-X-Y`文件中的'Y', 作为一个数组返回.

我们在RPC:CommitTask 中附加上述的信息给主机, 然后在CommitOK中处理它, 当RPC消息来源是执行map任务的worker的时候, 我们需要根据其生成的中间文件修改reduce metadata中的`Filename`(现在我们需要把这一个数据修改为`[]string`)

测试
```sh
go build -buildmode=plugin ../mrapps/wc.go

go run mrcoordinator.go test1.txt test2.txt

go run mrworker.go wc.so
```
```
=======
其中`pg-*.txt`是一个shell通配符, 匹配当前目录下所有以`pg-`开头的文件, `wc.so`则是一个**plugin**: 定义了**MapReduce**中`map()`和`reduce()`函数的具体内容, 在这里被`loadPlugin(os.Args[1])`从中加载具体的Map和Reduce函数, 返回`mapf`和`reducef`, 也就是这两个函数.

```go
intermediate := []mr.KeyValue{}
for _, filename := range os.Args[2:] {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()
	kva := mapf(filename, string(content))
	intermediate = append(intermediate, kva...)
}
```

需要在意的是`kva...`这里的语法, 这是`切片展开操作`, 将kva的内容逐个追加,而不是将kva作为一个元素追加到`intermediate`中; 这里结束后`intermediate`中记录的内容如下样式:

```
[
	{"hello", "1"},
	{"world", "1"},
	{"goodbye", "1"},
	{"world", "1"}
]
```

```go
sort.Sort(ByKey(intermediate))
oname := "mr-out-0"
ofile, _ := os.Create(oname)
```

接下来的操作和真正的MapReduce有很大的区别, 真正的MapReduce中, intermediate的数据会被分区到N*M个桶中, 而这里只是简单的存储在一个切片中.

然后对intermediate切片进行按key排序, 然后创建一个叫`mr-out-0`的文件, 然后返回一个文件句柄`ofile`, 准备进行reduce tasks

```go
i := 0
for i < len(intermediate) {
	j := i + 1
	for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
		j++
	}
	values := []string{}
	for k := i; k < j; k++ {
		values = append(values, intermediate[k].Value)
	}
	output := reducef(intermediate[i].Key, values)

	// this is the correct format for each line of Reduce output.
	fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

	i = j
}
ofile.Close()
```

对intermediate中每一个key(单词)都调用reduce函数, 不过在这之前要把传给reduce的参数处理出来:`key`和`values`, 举个例子就容易理解了:

```
// 排序后
intermediate = [
	{"Sakiko", "1"},
	{"Sakiko", "1"},
	{"Uika", "1"},
	{"Uika", "1"}
]
// 因为已经是排好序了, 所以文件中多次出现的单词产生的键值对会相邻
key="Sakiko"
values = [
	{"1"},
	{"1"}
]
```

将reduce后的结果按照正确的格式写入到文件句柄`ofile`管理的文件中.

至此, 我们了解了一个简单的**MapReduce**的程序是如何运行的, 可以看到它仅仅是一个进程, 没有并行以及分布式的运算, 我们的任务就是改进它!
>>>>>>> 4281666d7c582b8d27497ed15ad6fd45575a9477
