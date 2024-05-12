package kvsrv

import (
	"crypto/rand"
	"math/big"

	"6.5840/labrpc"
)

type Clerk struct {
	server *labrpc.ClientEnd
}

// 随机生成一个64位的数字, 重复的概率微乎其微,1e-19的概率
func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(server *labrpc.ClientEnd) *Clerk {
	ck := &Clerk{
		server: server,
	}
	return ck
}

// Get()操作
func (ck *Clerk) Get(key string) string {

	args := GetArgs{
		Key:   key,
		OpReg: nrand(),
	}
	reply := GetReply{}

	for !ck.server.Call("KVServer.Get", &args, &reply) {
	}

	// 发送清除缓存消息
	clearArgs := ClearArgs{
		OpReg: args.OpReg,
	}
	clearReply := ClearReply{}

	for !ck.server.Call("KVServer.Clear", &clearArgs, &clearReply) {
	}

	return reply.Value
}

// Put()或Append()操作
func (ck *Clerk) PutAppend(key string, value string, op string) string {

	args := PutAppendArgs{
		Key:   key,
		Value: value,
		OpReg: nrand(),
	}
	reply := PutAppendReply{}

	for !ck.server.Call("KVServer."+op, &args, &reply) {
	}

	// 发送清除缓存消息
	clearArgs := ClearArgs{
		OpReg: args.OpReg,
	}
	clearReply := ClearReply{}

	for !ck.server.Call("KVServer.Clear", &clearArgs, &clearReply) {
	}

	return reply.Value
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}

func (ck *Clerk) Append(key string, value string) string {
	return ck.PutAppend(key, value, "Append")
}
