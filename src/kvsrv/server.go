package kvsrv

import (
	"log"
	"sync"
	"time"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type KVServer struct {
	mu      sync.Mutex
	data    map[string]string
	OpCache map[int64]string
}

// 为了放置内存爆, 每次操作结束会清理该缓存值
func (kv *KVServer) Clear(args *ClearArgs, reply *ClearReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	delete(kv.OpCache, args.OpReg)
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	key := args.Key
	//opReg := args.OpReg

	kv.mu.Lock()
	defer kv.mu.Unlock()

	// value, ok := kv.OpCache[opReg]

	// // 滤去重复rpc
	// if ok {
	// 	reply.Value = value
	// 	return
	// }

	// // 缓存此次的rpc返回值
	// kv.OpCache[opReg] = kv.data[key]
	reply.Value = kv.data[key]
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	key := args.Key
	val := args.Value
	opReg := args.OpReg

	kv.mu.Lock()
	defer kv.mu.Unlock()

	value, ok := kv.OpCache[opReg]

	// 滤去重复rpc
	if ok {
		reply.Value = value
		return
	}

	// 缓存此次的rpc返回值
	kv.OpCache[opReg] = kv.data[key]
	reply.Value = kv.data[key]

	// 完成Put的工作
	kv.data[key] = val
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	key := args.Key
	val := args.Value
	opReg := args.OpReg

	kv.mu.Lock()
	defer kv.mu.Unlock()

	value, ok := kv.OpCache[opReg]

	// 滤去重复rpc
	if ok {
		reply.Value = value
		return
	}

	// 缓存此次的rpc返回值
	kv.OpCache[opReg] = kv.data[key]
	reply.Value = kv.data[key]

	// 完成Append的工作
	kv.data[key] += val
}

// 定期清理缓存线程(会导致一些操作丢失幂等性保证)
func (kv *KVServer) ClearClients(t int) {
	// 每t毫秒清理一次
	for {
		kv.mu.Lock()
		for k := range kv.OpCache {
			delete(kv.OpCache, k)
		}
		kv.mu.Unlock()
		time.Sleep(time.Duration(t) * time.Millisecond)
	}
}

func StartKVServer() *KVServer {
	//fmt.Print("KVServer")
	kv := &KVServer{
		mu:      sync.Mutex{},
		data:    make(map[string]string),
		OpCache: make(map[int64]string),
	}

	// 开启定期清理rpc缓存
	// go kv.ClearClients(500)

	return kv
}
