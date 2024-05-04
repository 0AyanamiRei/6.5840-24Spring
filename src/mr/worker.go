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
	fmt.Println("do....")

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
			num := ihash(kv.Value) % nReduce
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
			tempfile := fmt.Sprintf("mr-%d-%d.txt", task.TaskNum, i)

			// 创建临时文件"tempfile"
			if file, err = os.Create(tempfile); err != nil {
				log.Fatalf("cannot create %v", tempfile)
			}
			// 使用go的json包写入临时文件"tempfile"
			enc := json.NewEncoder(file)
			for _, kv := range intermediates[i] {
				if err = enc.Encode(&kv); err != nil {
					log.Fatalf("cannot encode %v", kv)
				}
			}
			// 关闭临时文件
			file.Close()
		}
	}

	fmt.Println("over!")
	return map2reduce
}

func doReduce(task tasksMetadata,
	reducef func(string, []string) string) {

	fmt.Println("do....")
	oname := fmt.Sprintf("mr-out-%d", task.TaskNum)
	ofile, _ := os.Create(oname)

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
		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)
		i = j
	}
	ofile.Close()
	fmt.Println("over!")
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

	for {
		task, jobstate, nReduce := AskTasks() // 请求任务
		if jobstate == JOBDONE {
			break
		} else if jobstate == RPCERROR {
			log.Fatal("RPCERROR")
		}

		debug_worker(task)
		intermediates := doTask(task, nReduce, mapf, reducef) // 执行任务
		CommitTask(task, intermediates)                       // 提交任务
	}
	fmt.Println("Worker关闭, JOB完成!")
}

func AskTasks() (tasksMetadata, int, int) {
	args := ExampleArgs{}
	reply := ExampleReply{}
	if ok := call("Coordinator.Task", &args, &reply); !ok {
		log.Fatal("call to Coordinator.Task failed")
	}
	return reply.Task, reply.JobState, reply.NReduce
}

func CommitTask(task tasksMetadata, intermediates []int) {
	args := ExampleArgs{
		Task:          task,
		Intermediates: intermediates,
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
