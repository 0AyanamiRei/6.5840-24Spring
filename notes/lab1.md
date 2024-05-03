# 解读函数

## wc.go

`wc.go`文件定义了**MapReduce**程序的`Map`和`Reduce`函数:

1. `Map(filename, contents)`, 两个参数分别是`文件名`和`文件内容`, 然后将文件内容分割成单词, 每个单词生成一个键值对:`<word, 1>`

2. `Reduce(key, values[])`, 第一个参数key即为一个单词, 然后values列表在这里实际就是[1,1,1...], 其长度就代表了这个单词出现在文件中的次数.

我们来具体看看代码,主要是想了解下不同文件之间传递了哪些数据结构.

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

`mr.KeyValue`就是上面在文件`worker.go`中定义的结构体.


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
	JobDone int           // 0:Job未做完 333:Map&Reduce任务完成
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

我们添加一对Rpc: `AskDone`和`Done`, worker向主机查询是否JOB完成, 从而来退出循环关闭程序.

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