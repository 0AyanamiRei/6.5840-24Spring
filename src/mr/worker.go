package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"
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

func doMap(task tasksMetadata,
	nReduce int,
	mapf func(string, string) []KeyValue) []int {

	var map2reduce []int
	var file *os.File
	var err error
	var content []byte
	intermediates := make([][]KeyValue, nReduce)

	// 读取文件调用mapf并分区存储到mf-X-Y文件
	for _, filename := range task.Filename {
		// 打开文件"filename"
		if file, err = os.Open(filename); err != nil {
			log.Fatalf("cannot open %v", filename)
		}
		// 读取"filename"的内容
		if content, err = ioutil.ReadAll(file); err != nil {
			log.Fatalf("cannot read %v", filename)
		}
		// 关闭"filename"
		file.Close()

		// 调用mapf并按键排序
		kva := mapf(filename, string(content))

		// 将临时k/v分区存储: MapNum-num
		for _, kv := range kva {
			num := ihash(kv.Key) % nReduce
			intermediates[num] = append(intermediates[num], kv)
		}

		// 为每个分区创建文件 mr-MapNum-i.txt
		for i := 0; i < nReduce; i++ {
			if len(intermediates[i]) == 0 {
				continue
			}
			// 追踪这个Map任务产生了哪些临时文件
			map2reduce = append(map2reduce, i)

			// 临时文件名格式化"tempfile"
			tempfile := fmt.Sprintf("temp-%d.txt", i)

			// 创建临时文件"tempfile"
			if file, err = os.CreateTemp("", tempfile); err != nil {
				log.Fatalf("cannot create %v", tempfile)
			}
			defer os.Remove(file.Name())

			// 使用go的json包写入临时文件"tempfile"
			enc := json.NewEncoder(file)
			for _, kv := range intermediates[i] {
				if err = enc.Encode(&kv); err != nil {
					log.Fatalf("cannot encode %v", kv)
				}
			}
			// 关闭临时文件
			file.Close()

			// 原子重名最终文件
			outfile := fmt.Sprintf("mr-%d-%d.txt", task.TaskNum, i)
			if err := os.Rename(file.Name(), outfile); err != nil {
				log.Fatal(err)
			}
		}
	}

	return map2reduce
}

func doReduce(task tasksMetadata,
	reducef func(string, []string) string) {
	var file *os.File
	var err error
	var intermediate []KeyValue

	for _, filename := range task.Filename {
		if file, err = os.Open(filename); err != nil {
			log.Fatalf("cannot open %v", filename)
		}
		dec := json.NewDecoder(file)
		// 暂时把文件的内容读到intermediate中
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err == io.EOF {
				break
			} else if err != nil {
				log.Fatalf("cannot decode %v", kv)
			}
			intermediate = append(intermediate, kv)
		}
		file.Close()
	}

	// 创建临时文件"tempfile"
	tempfile := fmt.Sprintf("temp-%d", task.TaskNum)
	file, _ = os.CreateTemp("", tempfile)
	defer os.Remove(file.Name())

	// 模仿sequential中的reduce
	sort.Sort(ByKey(intermediate))
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
		fmt.Fprintf(file, "%v %v\n", intermediate[i].Key, output)
		i = j
	}

	file.Close()
	outfile := fmt.Sprintf("mr-out-%d", task.TaskNum)

	if err = os.Rename(file.Name(), outfile); err != nil {
		log.Fatal(err)
	}

}

func debug_worker(task tasksMetadata) {
	if task.TaskType == REDUCE {
		fmt.Printf("(Rpc数据): 第%d号REDUCE任务,任务状态:%d,对象文件名:%s \n",
			task.TaskNum, task.TaskState, task.Filename)
	} else {
		fmt.Printf("(Rpc数据): 第%d号MAP任务,任务状态:%d,对象文件名:%s \n",
			task.TaskNum, task.TaskState, task.Filename)
	}
}

func doTask(
	task tasksMetadata,
	nReduce int,
	mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) []int {

	if task.TaskType == REDUCE {
		doReduce(task, reducef)
		return []int{}
	} else {
		return doMap(task, nReduce, mapf)
	}
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	//fmt.Printf("PID%d 开始执行\n", os.Getpid())

	// 启动心跳
	go SendHeartbeat(1)

	for {
		task, jobstate, nReduce := AskTasks() // 请求任务
		if jobstate == JOBDONE {
			break
		} else if jobstate == RPCERROR {
			//fmt.Printf("ID:%d 醒来了!\n", os.Getpid())
			continue
		}
		intermediates := doTask(task, nReduce, mapf, reducef) // 执行任务
		CommitTask(task, intermediates)                       // 提交任务
	}
	//fmt.Println("Worker关闭, JOB完成!")
}

// 请求任务
func AskTasks() (tasksMetadata, int, int) {
	args := ExampleArgs{
		WorkID: os.Getpid(),
	}
	reply := ExampleReply{}
	if ok := call("Coordinator.Task", &args, &reply); !ok {
		log.Fatal("call to Coordinator.Task failed")
	}
	return reply.Task, reply.JobState, reply.NReduce
}

// 提交任务
func CommitTask(task tasksMetadata, intermediates []int) {
	args := ExampleArgs{
		Task:          task,
		Intermediates: intermediates,
		WorkID:        os.Getpid(),
	}
	reply := ExampleReply{}

	if ok := call("Coordinator.CommitOK", &args, &reply); !ok {
		log.Fatal("call to Coordinator.CommitOK failed")
	}
}

// 发送心跳消息
func SendHeartbeat(HeartbeatInterval time.Duration) {
	args := ExampleArgs{
		WorkID: os.Getpid(),
	}
	reply := ExampleReply{}
	for {
		// 间隔HeartbeatInterval秒向主机汇报一次
		if ok := call("Coordinator.Heartbeat", &args, &reply); !ok {
			log.Fatal("call to Coordinator.Heartbeat failed")
		}
		//fmt.Println("heart!")
		time.Sleep(HeartbeatInterval * time.Second)
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
