package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

const IDLE = 0
const INPROC = 1
const COMPLETED = 2

const MAP = 0
const REDUCE = 1

const JOBDONE = 1
const JOBDOING = 0
const RPCERROR = 2

type tasksMetadata struct {
	TaskType  int      // 任务类型 0:Map,1:Reduce
	TaskNum   int      // 任务编号
	TaskState int      // 任务状态 0:IDLE,1:INPROC,2:COMPLETED
	Filename  []string // 读取文件名
}

type Coordinator struct {
	MapTasks    []tasksMetadata // 管理Map任务
	ReduceTasks []tasksMetadata // 管理Reduce任务
	NReduce     int             // 整个JobReduce任务数量
	NMapped     int             // 已完成的Map任务数量
	Nfile       int             // 整个Job读取的文件数量(Map任务数量)
}

func (c *Coordinator) debug_master(tasktype, num int) {
	if tasktype == REDUCE {
		fmt.Printf("(Master数据): 第%d号Reduce任务,任务状态:%d,对象文件名:%s\n",
			num, c.ReduceTasks[num].TaskState, c.ReduceTasks[num].Filename)
	} else {
		fmt.Printf("(Master数据): 第%d号Map任务,任务状态:%d,对象文件名:%s\n",
			num, c.MapTasks[num].TaskState, c.MapTasks[num].Filename)
	}
}

// 分配任务
func (c *Coordinator) Task(args *ExampleArgs, reply *ExampleReply) error {
	var task *tasksMetadata
	var num int
	Task := c.MapTasks
	tasktype := MAP
	if c.NMapped == c.Nfile {
		Task = c.ReduceTasks
		tasktype = REDUCE
	}

	for i, x := range Task {
		if x.TaskState == IDLE {
			task = &Task[i]
			num = i
			break
		}
	}

	// 正常来说Job已完成
	if task == nil {
		reply.JobState = JOBDONE
		// 保守地确认所有tasks是否完成
		for _, x := range c.ReduceTasks {
			if x.TaskState != COMPLETED {
				reply.JobState = RPCERROR
				return nil
			}
		}
		for _, x := range c.MapTasks {
			if x.TaskState != COMPLETED {
				reply.JobState = RPCERROR
				return nil
			}
		}
		return nil
	}

	// 修改task的元数据
	task.TaskType = tasktype
	task.TaskNum = num
	task.TaskState = INPROC

	// 返回分配好的task
	reply.NReduce = c.NReduce
	reply.Task = *task
	reply.JobState = JOBDOING

	//debug
	c.debug_master(tasktype, num)

	return nil
}

// 处理结束的任务
func (c *Coordinator) CommitOK(args *ExampleArgs, reply *ExampleReply) error {
	num := args.Task.TaskNum
	Task := c.MapTasks

	if args.Task.TaskType == REDUCE {
		Task = c.ReduceTasks
		c.NReduce--
	} else {
		fmt.Println(args.Intermediates)
		for _, x := range args.Intermediates {
			filename := fmt.Sprintf("mr-%d-%d.txt", num, x)
			c.ReduceTasks[x].Filename = append(c.ReduceTasks[x].Filename, filename)
		}
		c.NMapped++
	}
	Task[num].TaskState = COMPLETED
	return nil
}

// 初始化Coordinator实例
func initCoordinator(files []string, nReduce int) Coordinator {
	c := Coordinator{
		MapTasks:    make([]tasksMetadata, 0),
		ReduceTasks: make([]tasksMetadata, 0),
		NReduce:     nReduce,
		NMapped:     0,
		Nfile:       len(files),
	}

	// 初始化MapTasks
	for i, filename := range files {
		taskMeta := tasksMetadata{
			TaskType:  MAP,
			TaskNum:   i,
			TaskState: IDLE,
			Filename:  []string{filename},
		}
		c.MapTasks = append(c.MapTasks, taskMeta)
	}
	// 初始化ReduceTasks
	for i := 0; i < nReduce; i++ {
		taskMeta := tasksMetadata{
			TaskType:  REDUCE,
			TaskNum:   i,
			TaskState: IDLE,
			Filename:  []string{},
		}
		c.ReduceTasks = append(c.ReduceTasks, taskMeta)
	}
	return c
}

// 开启一个监听线程
func (c *Coordinator) server() {
	rpc.Register(c)  // 注册一个RPC服务, 其公开方法可以通过RPC调用
	rpc.HandleHTTP() // 将RPC服务绑定到HTTP协议上
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock() // 获取一个Unix域套接字的名称
	os.Remove(sockname)           // 删除可能存在的旧 Unix 域套接字
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil) // 启动一个HTTP服务器接收请求
}

// 查询Job是否完成
func (c *Coordinator) Done() bool {
	ret := true

	if c.NMapped != c.Nfile || c.NReduce != 0 {
		ret = false
		return ret
	}
	for _, task := range c.MapTasks {
		if task.TaskState != COMPLETED {
			ret = false
			return ret
		}
	}
	for _, task := range c.ReduceTasks {
		if task.TaskState != COMPLETED {
			ret = false
			return ret
		}
	}

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := initCoordinator(files, nReduce)
	c.server()
	return &c
}
