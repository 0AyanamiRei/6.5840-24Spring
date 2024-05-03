package mr

import (
	"errors"
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

const JOBDONE = 333

type tasksMetadata struct {
	TaskType  int    // 任务类型 0:Map,1:Reduce
	TaskNum   int    // 任务编号
	TaskState int    // 任务状态 0:IDLE,1:INPROC,2:COMPLETED
	Filename  string // 读取文件名
}

type Coordinator struct {
	MapTasks    []tasksMetadata // 管理Map任务
	ReduceTasks []tasksMetadata // 管理Reduce任务
	NReduce     int             // 整个JobReduce任务数量
	NMapped     int             // 已完成的Map任务数量
	Nfile       int             // 整个Job读取的文件数量(Map任务数量)
}

// Your code here -- RPC handlers for the worker to call.

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

	if task == nil {
		for i, x := range c.ReduceTasks {
			fmt.Println(i, x)
		}
		return errors.New("no idle task found")
	}

	// 修改task的元数据
	task.TaskType = tasktype
	task.TaskNum = num
	task.TaskState = INPROC

	reply.Task = *task

	// 调试信息
	if tasktype == REDUCE {
		fmt.Printf("(Master数据): 第%d号Reduce任务,任务状态:%d,对象文件名:%s\n",
			num, c.ReduceTasks[num].TaskState, c.ReduceTasks[num].Filename)
	} else {
		fmt.Printf("(Master数据): 第%d号Map任务,任务状态:%d,对象文件名:%s\n",
			num, c.MapTasks[num].TaskState, c.MapTasks[num].Filename)
	}

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
		c.NMapped++
	}
	Task[num].TaskState = COMPLETED
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
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
			Filename:  filename,
		}
		c.MapTasks = append(c.MapTasks, taskMeta)
	}
	// 初始化ReduceTasks
	for i := 0; i < nReduce; i++ {
		taskMeta := tasksMetadata{
			TaskType:  REDUCE,
			TaskNum:   i,
			TaskState: IDLE,
			Filename:  "CC",
		}
		c.ReduceTasks = append(c.ReduceTasks, taskMeta)
	}

	c.server()
	return &c
}
