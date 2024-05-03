package mr

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

type ByKey []KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

func doMap() {
	fmt.Println("do....")
	fmt.Println("over!")
}

func doReduce() {
	fmt.Println("do....")
	fmt.Println("over!")
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	for {
		task := AskTasks()
		// do task
		if task.TaskType == REDUCE {
			fmt.Printf("(Rpc数据): 第%d号REDUCE任务,任务状态:%d,对象文件名:%s \n",
				task.TaskNum, task.TaskState, task.Filename)
			doReduce()
		} else {
			fmt.Printf("(Rpc数据): 第%d号MAP任务,任务状态:%d,对象文件名:%s \n",
				task.TaskNum, task.TaskState, task.Filename)
			doMap()
		}
		CommitTask(task)
	}
}

func AskTasks() tasksMetadata {
	args := ExampleArgs{}
	reply := ExampleReply{}
	if ok := call("Coordinator.Task", &args, &reply); !ok {
		log.Fatal("call to Coordinator.Task failed")
	}
	return reply.Task
}

func CommitTask(task tasksMetadata) {
	args := ExampleArgs{
		Task: task,
	}
	reply := ExampleReply{}

	if ok := call("Coordinator.CommitOK", &args, &reply); !ok {
		log.Fatal("call to Coordinator.CommitOK failed")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
