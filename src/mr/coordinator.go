package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

const IDLE = 0
const INPROC = 1
const COMPLETED = 2

const MAP = 0
const REDUCE = 1

const JOBDONE = 1
const JOBDOING = 0
const RPCERROR = 2

const FREE = 0
const WORK = 1
const CRASH = 2

type WorkerStatus struct {
	LastHeartbeat time.Time
	TaskType      int
	TaskNum       int
	Status        int
	Mu            sync.Mutex // worker锁
}

type tasksMetadata struct {
	TaskType  int      // 任务类型 0:Map,1:Reduce
	TaskNum   int      // 任务编号
	TaskState int      // 任务状态 0:IDLE,1:INPROC,2:COMPLETED
	Filename  []string // 读取文件名
}

type Coordinator struct {
	Mu          sync.Mutex            // 加锁, 必须加大锁
	Cond        *sync.Cond            // RPCERROR时阻塞RPC处理程序
	MapTasks    []tasksMetadata       // 管理Map任务
	ReduceTasks []tasksMetadata       // 管理Reduce任务
	NReduce     int                   // 整个JobReduce任务数量
	NMapped     int                   // 已完成的Map任务数量
	Nfile       int                   // 整个Job读取的文件数量(Map任务数量)
	Workers     map[int]*WorkerStatus // 管理workers的状态
	Timeout     time.Duration         // 超过此事件认定worker崩溃
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

// 接收心跳消息并更新计时
func (c *Coordinator) Heartbeat(args *ExampleArgs, reply *ExampleReply) error {
	// 更新状态
	c.Mu.Lock()
	pid := args.WorkID
	if c.Workers[pid] == nil {
		c.Workers[pid] = &WorkerStatus{}
	}
	c.Mu.Unlock()

	c.Workers[pid].Mu.Lock()
	defer c.Workers[pid].Mu.Unlock()
	c.Workers[pid].LastHeartbeat = time.Now()
	//fmt.Println("worker=", args.WorkID, " time=", c.Workers[args.WorkID].LastHeartbeat,
	//	" 状态=", c.Workers[args.WorkID].Status)
	return nil
}

// 分配任务
func (c *Coordinator) Task(args *ExampleArgs, reply *ExampleReply) error {

	c.Mu.Lock()

	pid := args.WorkID
	if c.Workers[pid] == nil {
		c.Workers[pid] = &WorkerStatus{}
	}

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

	// 要具体细分是因为Map阶段未完成还是因为整个Job完成导致的nil
	if task == nil {
		reply.JobState = JOBDONE

		if c.NMapped != c.Nfile || c.NReduce != 0 {
			//fmt.Printf("ID:%d 陷入沉睡!\n", args.WorkID)
			c.Cond.Wait()
			reply.JobState = RPCERROR
		}

		c.Mu.Unlock()
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

	c.Mu.Unlock()

	// 修改Worker信息
	c.Workers[pid].Mu.Lock()
	c.Workers[pid].Status = WORK
	c.Workers[pid].TaskNum = num
	c.Workers[pid].TaskType = tasktype
	c.Workers[pid].Mu.Unlock()

	return nil
}

// 处理结束的任务
func (c *Coordinator) CommitOK(args *ExampleArgs, reply *ExampleReply) error {

	c.Mu.Lock()

	pid := args.WorkID
	num := args.Task.TaskNum
	Task := c.MapTasks

	if args.Task.TaskType == REDUCE {
		Task = c.ReduceTasks
		c.NReduce--
	} else {
		for _, x := range args.Intermediates {
			filename := fmt.Sprintf("mr-%d-%d.txt", num, x)
			c.ReduceTasks[x].Filename = append(c.ReduceTasks[x].Filename, filename)
		}
		c.NMapped++
	}
	Task[num].TaskState = COMPLETED

	c.Workers[pid].Mu.Lock()
	c.Workers[args.WorkID].Status = FREE
	c.Workers[pid].Mu.Unlock()

	// 视情况唤醒阻塞的Task线程
	if args.Task.TaskType == MAP && c.NMapped == c.Nfile {
		c.Cond.Broadcast()
	}
	if args.Task.TaskType == REDUCE && c.NReduce == 0 {
		c.Cond.Broadcast()
	}

	c.Mu.Unlock()

	return nil
}

// 初始化Coordinator实例
func initCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		MapTasks:    make([]tasksMetadata, 0),
		ReduceTasks: make([]tasksMetadata, 0),
		NReduce:     nReduce,
		NMapped:     0,
		Nfile:       len(files),
		Workers:     make(map[int]*WorkerStatus),
		Timeout:     4,
	}

	c.Mu = sync.Mutex{}
	c.Cond = sync.NewCond(&c.Mu)

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
	return &c
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

	c.Mu.Lock()

	if c.NMapped != c.Nfile || c.NReduce != 0 {
		ret = false
		c.Mu.Unlock()
		return ret
	}

	c.Mu.Unlock()
	return ret
}

// 定期检查每个Workers
func (c *Coordinator) Heart() {
	for {
		time.Sleep(2 * time.Second)
		now := time.Now()
		for pid, worker := range c.Workers {

			c.Workers[pid].Mu.Lock()

			if worker.Status == CRASH {
				c.Workers[pid].Mu.Unlock()
				continue
			}

			if now.Sub(worker.LastHeartbeat) > c.Timeout*time.Second {

				worker.Status = CRASH
				//fmt.Printf("Worker %d has crashed\n", pid)

				if worker.TaskType == REDUCE {
					c.ReduceTasks[worker.TaskNum].TaskState = IDLE
				} else {
					c.MapTasks[worker.TaskNum].TaskState = IDLE
				}
				// 必须要唤醒
				c.Cond.Broadcast()
			}

			c.Workers[pid].Mu.Unlock()
		}
	}
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := initCoordinator(files, nReduce)
	c.server()
	go c.Heart()
	return c
}
