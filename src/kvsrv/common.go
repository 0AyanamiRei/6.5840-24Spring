package kvsrv

// Put or Append
type PutAppendArgs struct {
	Key   string
	Value string
	OpReg int64
}

type PutAppendReply struct {
	Value string
}

type GetArgs struct {
	Key   string
	OpReg int64
}

type GetReply struct {
	Value string
}

type ClearArgs struct {
	OpReg int64
}

type ClearReply struct {
	OpReg int64
}
