package raft

import (
	//	"bytes"

	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labgob"
	"6.5840/labrpc"
)

//************************************************************************************
// 定义数据结构
//************************************************************************************

const (
	// 服务器状态
	CANDIDATE int = 2
	LEADER    int = 1
	FOLLOWER  int = 0

	// DEBUG
	DEBUG           bool = false
	DEBUG_Info      bool = false
	DEBUG_Vote      bool = true
	DEBUG_Heartbeat bool = false
	DEBUG_Aped      bool = true
	DEBUG_VoteRpc   bool = false
	DEBUG_HeartRpc  bool = false
	DEBUG_ApedRpc   bool = false
	DEBUG_Persist   bool = false
	DEBUG_Snap      bool = true
)

var STATE = []string{"F", "L", "C"}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// 按照论文图2所示定义以下状态
	Log         []Log // 日志
	CurrentTerm int   // 任期
	VotedFor    int   // 已投candidate的ID 真的需要吗?
	CommitIndex int   // (全局)已提交的日志(大多数服务器写入Log后就算作已提交)
	LastApplied int   // (全局)已执行的日志(已提交的日志分为已执行和未执行)
	NextIndex   []int // (虚拟)发送到该服务器的下一个日志条目的索引
	MatchIndex  []int // (虚拟)已知的已经复制到该服务器的最高日志条目的索引

	// 自定义状态
	LastRPC   time.Time // 接收心跳消息的时间
	State     int       // 确认是否为领导人
	LogLength int       // (虚拟)用变量维护日志长度

	// 提供给应用的接口
	ApplyCh   chan ApplyMsg
	ApplyCond *sync.Cond

	// 快照
	SnapshotData      []byte
	LastIncludedIndex int
	LastIncludedTerm  int
}

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int // (全局)

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type Log struct {
	LogTerm int
	Command interface{}
}

type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term         int
	CandidateId  int
	LastLogIndex int // (虚拟)
	LastLogTerm  int
}

type RequestVoteReply struct {
	// Your data here (3A).
	Term        int  // 参与投票的服务器自己记录的任期
	VoteGranted bool // true表示投给该candidate
}

// 追加日志 + 心跳
type AppendEntriesArgs struct {
	Term         int   // leader的任期
	LeaderId     int   // leader的网络标识符
	PrevLogIndex int   // (全局)简单的一致性检查
	PrevLogTerm  int   // 简单的一致性检查
	Entries      []Log // 日志内容(记得附加此前任期的日志内容)
	LeaderCommit int   // (全局)leader追踪的已提交日志下标
}

type AppendEntriesReply struct {
	Term    int  // follower的任期号, 以便leader更新
	Success bool // 日志添加是否成功
	// fast bakeup
	XTerm  int // 冲突的任期号, -1表示该日志条目不存在
	XLen   int // (全局)如果XTerm=-1, 记录leader应该追加日志的起始下标
	XIndex int // (全局)如果XTerm!=-1, 记录该任期第一个日志条目的下标
}

type SnapshotArgs struct {
	Term              int
	LeaderId          int
	LastIncludedIndex int
	LastIncludedTerm  int
	Snapdata          []byte
}

type SnapshotRelpay struct {
	Term int
}

//************************************************************************************
// Tools
//************************************************************************************

// 调用前持有rf.mu
func (rf *Raft) V2PIndex(idx int) int {
	return idx - rf.LastIncludedIndex
}

// 调用前持有rf.mu
func (rf *Raft) P2VIndex(idx int) int {
	return idx + rf.LastIncludedIndex
}

func Min(a int, b int) int {
	if a > b {
		return b
	} else {
		return a
	}
}

func Max(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.CurrentTerm, rf.State == LEADER
}

// 调用前需持有锁rf.mu
//
//	需要做持久化的状态: (1)CurrentTerm, (2)VotedFor, (3)Log
func (rf *Raft) persist() {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.CurrentTerm)
	e.Encode(rf.VotedFor)
	e.Encode(rf.Log)
	e.Encode(rf.LastIncludedIndex)
	e.Encode(rf.LastIncludedTerm)
	raftstate := w.Bytes()

	rf.persister.Save(raftstate, rf.SnapshotData)
}

// restore previously persisted state.
func (rf *Raft) readPersist(raftdata []byte, snapshot []byte) {
	// read RaftState
	if raftdata == nil || len(raftdata) < 1 { // bootstrap without any state?
		return
	}

	r := bytes.NewBuffer(raftdata)
	d := labgob.NewDecoder(r)

	var CurrentTerm int
	var VotedFor int
	var Log []Log
	var LastIncludedIndex int
	var LastIncludedTerm int

	if d.Decode(&CurrentTerm) != nil ||
		d.Decode(&VotedFor) != nil ||
		d.Decode(&Log) != nil ||
		d.Decode(&LastIncludedIndex) != nil ||
		d.Decode(&LastIncludedTerm) != nil {
		return
	} else {
		rf.CurrentTerm = CurrentTerm
		rf.VotedFor = VotedFor
		rf.Log = Log
		rf.LogLength = len(Log) + LastIncludedIndex
		rf.LastIncludedIndex = LastIncludedIndex
		rf.LastIncludedTerm = LastIncludedTerm
	}

	//read Snapshot State

	if snapshot == nil || len(snapshot) < 1 {
		return
	}

	rf.SnapshotData = snapshot
}

// 调用前不持有锁rf.mu, 调用后也不持有锁rf.mu
//
// 提供给上层的接口
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 修改LastIncludedIndex之前进行
	rf.LastIncludedTerm = rf.Log[rf.V2PIndex(index)].LogTerm
	rf.Log = rf.Log[rf.V2PIndex(index):]

	rf.LastIncludedIndex = index
	rf.LogLength = len(rf.Log) + rf.LastIncludedIndex
	rf.SnapshotData = snapshot

	if DEBUG_Snap {
		DPrintf("[SNAP] p%d(%v,%d)\" log[0:%d:%d) LastInclude(idx:%d,term:%d)\"\n", rf.me, STATE[rf.State], rf.CurrentTerm,
			rf.LastIncludedIndex, rf.LogLength,
			rf.LastIncludedIndex, rf.LastIncludedTerm)
	}

	rf.persist()
}

//************************************************************************************
// RPCs
//************************************************************************************

// 发送RPC消息
func (rf *Raft) SendRPC(to int, rpc string, args interface{}, reply interface{}) bool {
	return rf.peers[to].Call("Raft."+rpc, args, reply)
}

// leader发送快照给peers[to]
//
// 调用前不持有rf.mu, 调用后也不持有rf.mu
func (rf *Raft) SendSnapshot(to int, args SnapshotArgs) {
	reply := SnapshotRelpay{}

	if DEBUG_Snap {
		DPrintf("[SNAP] p%d(%v,%d)->p%d \"SendSnapshot  snap(idx:%d,term:%d) NextIdx[%d]=%d->%d\"\n",
			rf.me, STATE[rf.State], rf.CurrentTerm, to,
			rf.LastIncludedIndex, rf.LastIncludedTerm,
			to, rf.NextIndex[to], rf.LastIncludedIndex+1)
	}

	if !rf.SendRPC(to, "InstallSnapshot", &args, &reply) {
		return
	}

	rf.mu.Lock()

	//rpc回复过期
	if rf.CurrentTerm != args.Term {
		rf.mu.Unlock()
		return
	}

	// 任期检查
	if reply.Term > rf.CurrentTerm {
		rf.State = FOLLOWER
		//rf.VotedFor = -1
		rf.CurrentTerm = reply.Term

		rf.persist()
	}

	rf.NextIndex[to] = rf.LastIncludedIndex + 1
	rf.mu.Unlock()
}

// 快照处理程序
//
// 1: rpc任期检查
//
// 2: 更新计时器
//
// 3: 更新日志项
//
// 4: 修改自身配置项, 应用快照
//
// 调用前无锁 rf.mu, 调用后也不持有锁 rf.mu
func (rf *Raft) InstallSnapshot(args *SnapshotArgs, reply *SnapshotRelpay) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.CurrentTerm

	// 1
	if args.Term < rf.CurrentTerm {
		return
	}

	if args.Term > rf.CurrentTerm {
		rf.CurrentTerm = args.Term
		rf.VotedFor = -1 // why this code??
		rf.State = FOLLOWER
	}

	// 2
	rf.LastRPC = time.Now()

	// 3
	if rf.LastIncludedIndex <= args.LastIncludedIndex &&
		args.LastIncludedIndex <= rf.LogLength-1 &&
		rf.Log[rf.V2PIndex(args.LastIncludedIndex)].LogTerm == args.LastIncludedTerm {
		rf.Log = rf.Log[rf.V2PIndex(args.LastIncludedIndex):]
	} else {
		rf.Log = make([]Log, 0)
		rf.Log = append(rf.Log, Log{args.LastIncludedTerm, 0})
	}

	// 4
	rf.LastIncludedIndex = args.LastIncludedIndex
	rf.LastIncludedTerm = args.LastIncludedTerm
	rf.LastApplied = Max(rf.LastApplied, args.LastIncludedIndex)
	rf.CommitIndex = Max(rf.CommitIndex, args.LastIncludedIndex)
	rf.SnapshotData = args.Snapdata
	rf.LogLength = len(rf.Log) + rf.LastIncludedIndex

	msg := ApplyMsg{
		SnapshotValid: true,
		Snapshot:      args.Snapdata,
		SnapshotTerm:  args.LastIncludedTerm,
		SnapshotIndex: args.LastIncludedIndex,
	}

	rf.persist()

	if DEBUG_Snap {
		DPrintf("[SNAP] p%d(%v,%d) \"InstallSnap snap(idx:%d,term:%d) CMIT:%d APLY:%d\"\n",
			rf.me, STATE[rf.State], rf.CurrentTerm,
			rf.LastIncludedIndex, rf.LastIncludedTerm,
			rf.CommitIndex, rf.LastApplied)
	}

	rf.mu.Unlock()
	rf.ApplyCh <- msg
	rf.mu.Lock()
}

// 发起一轮领导人选举
//
// 1: 自增任期号
//
// 2: 投票给自己
//
// 3: 刷新计时器
//
// 4: 调用SendElection并行发送RequestVote RPCs给其他服务器
//
// 调用前不持有锁 rf.mu
func (rf *Raft) BeginElection() {
	rf.mu.Lock()
	// 1:
	rf.CurrentTerm++
	// 2:
	rf.VotedFor = rf.me
	var vote uint32 = 1

	rf.persist() // 持久化

	// 3:
	rf.LastRPC = time.Now()

	if rf.V2PIndex(rf.LogLength)-1 < 0 {
		DPrintf("[DEBUG] p%d(%v,%d) \"BeginElection snap(idx:%d,term:%d) LogLen:%d\"\n",
			rf.me, STATE[rf.State], rf.CurrentTerm,
			rf.LastIncludedIndex, rf.LastIncludedTerm, rf.LogLength)
	}

	args := RequestVoteArgs{
		Term:         rf.CurrentTerm,
		CandidateId:  rf.me,
		LastLogIndex: rf.LogLength - 1,
		LastLogTerm:  rf.Log[rf.V2PIndex(rf.LogLength)-1].LogTerm,
	}

	if DEBUG_Vote {
		DPrintf("[VOTE-%d] p%d(%v,%d): Vote Start\n",
			args.Term, rf.me, STATE[rf.State], rf.CurrentTerm)
	}

	rf.mu.Unlock()

	// 4:
	for i := range rf.peers {
		if i != rf.me {
			go rf.SendElection(i, args, &vote)
		}
	}
}

// 向peer[to]发送投票请求
//
// 1: 处理过期rpc回复
//
// 2: 得到投票
//
// 3: 常规RPC检查, 自身任期过期
// 调用前不持有锁 rf.mu
func (rf *Raft) SendElection(to int, args RequestVoteArgs, vote *uint32) {
	reply := RequestVoteReply{}

	if DEBUG_Vote {
		DPrintf("[VOTE-%d] p%d(%v,%d)->p%d \"Ask Vote\"\n",
			args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, to)
	}

	if !rf.SendRPC(to, "RequestVote", &args, &reply) {
		if DEBUG_VoteRpc {
			DPrintf("[VOTE-%d] p%d(%v,%d)->p%d \"RequestVote RPCs failed\"\n",
				args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, to)
		}
		return
	}

	// 1:
	rf.mu.Lock()
	if rf.CurrentTerm != args.Term {
		rf.mu.Unlock()
		return
	}
	rf.mu.Unlock()

	// 2:
	if reply.VoteGranted {
		// 得票
		if v := atomic.AddUint32(vote, 1); v > (uint32(len(rf.peers) / 2)) {
			// 当选leader
			if DEBUG_Vote {
				DPrintf("[VOTE-%d] p%d(%v,%d): Got %d votes\n",
					args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, *vote)
			}

			rf.mu.Lock()

			rf.State = LEADER
			// 初始化MatchIndex和NextIndex
			for i := range rf.peers {
				if i >= len(rf.NextIndex) {
					rf.NextIndex = append(rf.NextIndex, rf.LogLength)
				} else {
					rf.NextIndex[i] = rf.LogLength
				}
				if i >= len(rf.MatchIndex) {
					rf.MatchIndex = append(rf.MatchIndex, -1)
				} else {
					rf.MatchIndex[i] = -1
				}
			}

			if DEBUG_Info {
				DPrintf("[Info] p%d(%v,%d) \"CMIT(%d) APLY(%d) LogLen=%d, log[last]=%v NextIndex:%v MatchIndex:%v\"",
					rf.me, STATE[rf.State], rf.CurrentTerm, rf.CommitIndex,
					rf.LastApplied, rf.LogLength, rf.Log[rf.V2PIndex(rf.LogLength)-1], rf.NextIndex, rf.MatchIndex)
			}
			// 发一轮心跳宣言自己的身份
			go rf.HeartBeatLauncher()

		}
	}

	// 3:
	rf.mu.Lock()
	if !reply.VoteGranted && reply.Term > rf.CurrentTerm {
		rf.CurrentTerm = reply.Term
		rf.persist() // 持久化
		rf.State = FOLLOWER
	}
	rf.mu.Unlock()
}

// 投票请求处理程序
//
// 对投票的要求
// 0: rpc共通检查, 如果自己的任期落后, 更新接收方的任期号, 并且更新状态为FOLLOWER
// 1(任期落后): 如果自己的任期号比候选人的任期号更大, 则立刻拒绝
// 2(本轮已投): 投票人若本轮投过票, 则立刻拒绝
// 3(日志落后): 若自己的日志内容比候选人的日志更新, 则立刻拒绝  (新: 最后一条日志任期更大, 下标更大)
// 4: 如果投票成功, 那么需要重置选举超时计时器, 修改自身一系列状态;
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reason := 0 //1(2) 2(4) 3(8)

	reply.Term = rf.CurrentTerm
	reply.VoteGranted = false

	//0 任期更新
	if rf.CurrentTerm < args.Term {
		rf.State = FOLLOWER
		rf.CurrentTerm = args.Term
		rf.VotedFor = -1
	}

	//1: 任期过时
	if rf.CurrentTerm > args.Term {
		reason += 1 << 1
	}

	//2: 已投票 (解释为什么要rf.VotedFor != args.CandidateId)
	if rf.VotedFor != -1 && rf.VotedFor != args.CandidateId {
		reason += 1 << 2
	}

	// 较新的日志: (最后一个条目任期更大) || (最后一个条目任期相同&&下标更大)
	//3: 候选人日志不如自己新
	if !((args.LastLogTerm > rf.Log[rf.V2PIndex(rf.LogLength)-1].LogTerm) ||
		(args.LastLogTerm == rf.Log[rf.V2PIndex(rf.LogLength)-1].LogTerm &&
			args.LastLogIndex >= rf.LogLength-1)) {
		reason += 1 << 3
	}

	//4: 投票成功
	if reason == 0 {
		reply.VoteGranted = true
		rf.CurrentTerm = args.Term
		rf.VotedFor = args.CandidateId
		// 投票成功后重置定时器
		rf.LastRPC = time.Now()
	}

	// 持久化
	if reason == 0 || rf.CurrentTerm < args.Term {
		rf.persist()
	}

	if DEBUG_Vote {
		if reason == 0 {
			DPrintf("[VOTE-%d] p%d(%v,%d)->p%d \"OKKKK\"\n",
				args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, args.CandidateId)
		} else {
			DPrintf("[VOTE-%d] p%d(%v,%d)->p%d \"Refuse Vote:(%d)\"\n",
				args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, args.CandidateId, reason)
		}
	}
}

// 心跳发射器
//
// 领导人组织好心跳包准备发送给其他服务器
//
// 调用前获得锁 rf.mu, 调用后释放了锁 rf.mu
func (rf *Raft) HeartBeatLauncher() {
	// 更新自己
	rf.LastRPC = time.Now()
	// 提前复制资源, 以释放锁
	Term := rf.CurrentTerm
	LeaderId := rf.me
	LeaderCommit := rf.CommitIndex
	LastIncludedIndex := rf.LastIncludedIndex
	LastIncludedTerm := rf.LastIncludedTerm
	PrevLogIndex := make([]int, len(rf.peers))
	PrevLogTerm := make([]int, len(rf.peers))
	Entries := make([][]Log, len(rf.peers))

	for i := range rf.peers {
		if i == rf.me {
			continue
		}

		PrevLogIndex[i] = rf.NextIndex[i] - 1
		//允许等于0, rf.LastIncludedIndex/Term就是用于index=0时的一致性检查
		if rf.V2PIndex(rf.NextIndex[i])-1 < 0 { // SendSnapshot
			PrevLogTerm[i] = -1
		} else if rf.LogLength > rf.NextIndex[i] { // 追加日志
			PrevLogTerm[i] = rf.Log[rf.V2PIndex(rf.NextIndex[i])-1].LogTerm
			Entries[i] = make([]Log, len(rf.Log[rf.V2PIndex(rf.NextIndex[i]):]))
			copy(Entries[i], rf.Log[rf.V2PIndex(rf.NextIndex[i]):])
			if DEBUG_Aped {
				DPrintf("[APED] p%d(%v,%d)->p%d \"Aped Log[%d:%d) (-%d)\"\n",
					rf.me, STATE[rf.State], rf.CurrentTerm, i,
					rf.NextIndex[i], rf.LogLength, rf.LastIncludedIndex)
			}
		} else { // 心跳消息
			PrevLogTerm[i] = rf.Log[rf.V2PIndex(rf.NextIndex[i])-1].LogTerm
			Entries[i] = make([]Log, 0)
			if DEBUG_Heartbeat {
				DPrintf("[Heart(%d)] sent p%d(%s)->p%d\n", rf.CurrentTerm, rf.me, STATE[rf.State], i)
			}
		}
	}

	rf.mu.Unlock()

	// 并发发送心跳&追加日志消息
	for i := range rf.peers {
		if i == LeaderId {
			continue
		}

		if PrevLogTerm[i] != -1 { // SendHeartbeat
			args := AppendEntriesArgs{
				Term:         Term,
				LeaderId:     LeaderId,
				PrevLogIndex: PrevLogIndex[i],
				PrevLogTerm:  PrevLogTerm[i],
				LeaderCommit: LeaderCommit,
				Entries:      Entries[i],
			}
			go rf.SendHeartbeat(i, args)
		} else {
			args := SnapshotArgs{
				Term:              Term,
				LeaderId:          LeaderId,
				LastIncludedIndex: LastIncludedIndex,
				LastIncludedTerm:  LastIncludedTerm,
				Snapdata:          rf.SnapshotData, // 不需要锁也行
			}
			go rf.SendSnapshot(i, args)
		}
	}
}

// 心跳包发射函数
//
// 发送心跳包给peer[to]
//
// 调用前无锁 rf.mu, 调用后也不持有锁 rf.mu
func (rf *Raft) SendHeartbeat(to int, args AppendEntriesArgs) {
	reply := AppendEntriesReply{}

	// 由于网络问题导致RPC丢失 暂时不重发
	if !rf.SendRPC(to, "HeartbeatHandler", &args, &reply) {
		return
	}

	rf.mu.Lock()

	// rpc的回复过期,直接放弃
	if rf.CurrentTerm != args.Term {
		rf.mu.Unlock()
		return
	}

	// leader的任期过时导致失败
	if args.Term < reply.Term {
		if rf.CurrentTerm < reply.Term {
			rf.CurrentTerm = reply.Term
			rf.persist() // 持久化
		}
		rf.State = FOLLOWER
		rf.mu.Unlock()
		return
	}

	// 一致性检查+任期检查通过
	if reply.Success {
		// 助教提示如果设置nextIndex[i]-1或len(log)不安全
		rf.MatchIndex[to] = args.PrevLogIndex + len(args.Entries)
		rf.NextIndex[to] = rf.MatchIndex[to] + 1

		// 更新rf.CommitIndex = N(大多数服务器持有, 且任期等于当前任期的日志下标)
		if len(args.Entries) != 0 {
			for N := rf.LogLength - 1; N > rf.CommitIndex; N-- {
				count := 0 // 统计"大多数"
				for j := range rf.MatchIndex {
					if j == rf.me || (rf.MatchIndex[j] >= N && rf.Log[rf.V2PIndex(N)].LogTerm == rf.CurrentTerm) {
						count++
					}
				}
				// 更新rf.CommitIndex 需要在这里唤醒apply线程
				if count > len(rf.peers)/2 {
					if DEBUG_Aped {
						DPrintf("[CMIT] p%d(%v,%d) \"CMIT(%d->%d) APLY(%d->%d)\"\n",
							rf.me, STATE[rf.State], rf.CurrentTerm,
							rf.CommitIndex, N,
							rf.LastApplied, Max(rf.LastApplied, rf.LastIncludedIndex))
					}
					rf.CommitIndex = N
					rf.LastApplied = Max(rf.LastApplied, rf.LastIncludedIndex) // 安全性
					rf.ApplyCond.Broadcast()
					rf.mu.Unlock()
					return
				}
			}
		}
		rf.mu.Unlock()
		return
	}

	// 一致性检查不通过
	if !reply.Success {
		// 任期冲突 (这里采取了fast backup优化)
		if reply.XTerm != -1 {
			find := false
			for j := range rf.Log {
				// 找到该冲突任期在leader日志中的下标
				if rf.Log[j].LogTerm == reply.XTerm {
					rf.NextIndex[to] = rf.P2VIndex(j) + 1
					find = true
					break
				}
			}

			// leader日志中不存在该冲突任期
			if !find {
				rf.NextIndex[to] = reply.XIndex
			}
		} else { // 检查日志不存在
			rf.NextIndex[to] = reply.XLen
		}
		// 如果这里退回到的下标已经被快照覆盖, 那么发snapshot
		// 但是其实也可以等下一轮心跳的
		if rf.NextIndex[to] <= rf.LastIncludedIndex {
			args := SnapshotArgs{
				Term:              rf.CurrentTerm,
				LeaderId:          rf.me,
				LastIncludedIndex: rf.LastIncludedIndex,
				LastIncludedTerm:  rf.LastIncludedTerm,
				Snapdata:          rf.SnapshotData,
			}
			rf.mu.Unlock()
			go rf.SendSnapshot(to, args)
			return
		}

		rf.mu.Unlock()
	}
}

// 心跳处理程序
//
// 0: rpc共通检查, 如果自己的任期落后, 更新接收方的任期号, 并且更新状态为FOLLOWER
//
// 1(任期落后): 如果领导人的任期比自己小, 那么放弃这次心跳消息, 不重置选举计时器
//
// 2(一致性检查): -2a) 被检查日志条目不存在 -2b) 被检查日志条目任期冲突
//
// 调用前不持有锁 rf.mu, 调用后也不持有锁 rf.mu
func (rf *Raft) HeartbeatHandler(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	reply.Term = rf.CurrentTerm

	// 0: 更新自己的任期
	if args.Term > rf.CurrentTerm {
		rf.CurrentTerm = args.Term
		rf.State = FOLLOWER // why need this?
		rf.persist()        // 持久化
	}

	// 1:领导人任期落后
	if args.Term < rf.CurrentTerm {
		reply.Success = false
		if DEBUG_Aped {
			DPrintf("[TEMP] p%d(%v,%d) \"refuse for 1 %d<%d\"\n",
				rf.me, STATE[rf.State], rf.CurrentTerm,
				args.Term, rf.CurrentTerm)
		}
		return
	}

	//(除了rpc任期过期外都需要选举计时器)
	rf.LastRPC = time.Now()

	// 2a: 被检查日志条目不存在 (按全局索引比较
	if args.PrevLogIndex > rf.LogLength-1 {
		reply.XTerm = -1
		reply.XLen = rf.LogLength
		reply.Success = false

		// DEBUG
		DPrintf("[TEMP] p%d(%v,%d) \" 2a: %d>%d-1 XTerm=-1,XLen=%d\"\n",
			rf.me, STATE[rf.State], rf.CurrentTerm,
			args.PrevLogIndex, rf.LogLength,
			reply.XLen)
		return
	}

	// 思考这里tobeCheckIndex会<0吗 这意味着leader的快照更旧
	tobeCheckIndex := rf.V2PIndex(args.PrevLogIndex)
	tobeCheckTerm := rf.Log[tobeCheckIndex].LogTerm
	if tobeCheckIndex == 0 {
		tobeCheckTerm = rf.LastIncludedTerm
		rf.Log[0].LogTerm = rf.LastIncludedTerm //方便后面从0开始
	}

	// 2b: 被检查日志条目任期冲突
	if args.PrevLogTerm != tobeCheckTerm {
		reply.XTerm = tobeCheckTerm
		reply.Success = false
		// 找到该任期在日志第一次出现的下标
		for i := 0; i <= tobeCheckIndex; i++ {
			if rf.Log[i].LogTerm == tobeCheckTerm {
				reply.XIndex = rf.P2VIndex(i)
				break
			}
		}

		// 删除包括该冲突日志在内的后续所有日志
		if tobeCheckIndex == 0 {
			rf.Log = rf.Log[:1]
		} else {
			rf.Log = rf.Log[:tobeCheckIndex]
		}
		rf.LogLength = rf.P2VIndex(len(rf.Log))
		rf.persist()

		// DEBUG
		DPrintf("[TEMP] p%d(%v,%d) \" 2b: %d!=[%d+%d].%d XTerm=%d,XIndex=%d\"\n",
			rf.me, STATE[rf.State], rf.CurrentTerm,
			args.PrevLogTerm, tobeCheckIndex, rf.LastIncludedIndex, tobeCheckTerm,
			reply.XTerm, reply.XIndex)
		return
	}

	// 附加了日志(这里通过了一致性检查+任期检查)
	if len(args.Entries) != 0 {
		if DEBUG_Aped {
			DPrintf("[APED] p%d(%v,%d) \"CMIT(%d) APLY(%d) Add Log[%d:%d] log[last]:%v\"\n",
				rf.me, STATE[rf.State], rf.CurrentTerm, rf.CommitIndex, rf.LastApplied,
				args.PrevLogIndex, args.PrevLogIndex+len(args.Entries), args.Entries[len(args.Entries)-1])
		}
		// 在网络有故障时要注意这里
		rf.Log = append(rf.Log[:tobeCheckIndex+1], args.Entries...)
		rf.LogLength = rf.P2VIndex(len(rf.Log))
		rf.persist() // 持久化
	}

	reply.Success = true

	// 针对重连上来的leader
	// rf.State = FOLLOWER

	// AppendEntries RPC的第五条建议 需要唤醒apply线程
	if args.LeaderCommit > rf.CommitIndex {
		if DEBUG_Aped {
			DPrintf("[CMIT] p%d(%v,%d) \"CMIT(%d->%d) APLY(%d->%d)\"\n",
				rf.me, STATE[rf.State], rf.CurrentTerm,
				rf.CommitIndex, Min(args.LeaderCommit, rf.LogLength-1),
				rf.LastApplied, Max(rf.LastApplied, rf.LastIncludedIndex))
		}

		rf.CommitIndex = Min(args.LeaderCommit, rf.LogLength-1)
		rf.LastApplied = Max(rf.LastApplied, rf.LastIncludedIndex) // 保证LastApplied不小于LastIncludedIndex
		rf.ApplyCond.Broadcast()
		return
	}
}

// Client sent request to leader by this func: Client-->Leader
//
// 1: append to Leader's Log
// 2: Leader send this log entries
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.State != LEADER {
		return -1, -1, false
	}

	// 1:
	entry := Log{LogTerm: rf.CurrentTerm, Command: command}
	rf.Log = append(rf.Log, entry)
	rf.LogLength++

	rf.persist() // persist

	if DEBUG_Aped {
		DPrintf("[Start] p%d(%v,%d) \"Add Log:%v\"\n", rf.me, STATE[rf.State], rf.CurrentTerm, entry)
	}

	return rf.LogLength - 1, rf.CurrentTerm, true
}

//************************************************************************************
// Kill
//************************************************************************************

func (rf *Raft) Kill() {
	DPrintf("p%d was killed\n", rf.me)
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

//************************************************************************************
// Goroutine
//************************************************************************************

// Election线程
//
// 超时则调用BeginElection()发起一轮投票
func (rf *Raft) ElectionGoroutine() {
	for !rf.killed() {
		ms := 350 + (rand.Int63() % 200)
		time.Sleep(time.Duration(ms) * time.Millisecond)

		rf.mu.Lock()
		if rf.State == FOLLOWER && time.Since(rf.LastRPC) > time.Duration(ms)*time.Millisecond {
			rf.mu.Unlock()
			rf.BeginElection()
		} else {
			rf.mu.Unlock()
		}
	}
}

// HeartBeat线程
//
// 刷新其他服务器心跳接收时间+领导人新日志发送
func (rf *Raft) HeartBeatGoroutine(heartTime int) {
	cnt := 0
	for !rf.killed() {
		rf.mu.Lock()

		if cnt%3 == 0 && DEBUG_Info {
			DPrintf("[Info] p%d(%v,%d) \"CMIT(%d) APLY(%d) SnapIndex=%d LogLen=%d log[last]=%v NextIndex:%v MatchIndex:%v\"",
				rf.me, STATE[rf.State], rf.CurrentTerm, rf.CommitIndex,
				rf.LastApplied, rf.LastIncludedIndex, rf.LogLength,
				rf.Log[rf.V2PIndex(rf.LogLength)-1], rf.NextIndex, rf.MatchIndex)
		}

		if rf.State == LEADER {
			rf.HeartBeatLauncher()
		} else {
			rf.mu.Unlock()
		}

		time.Sleep(time.Duration(heartTime) * time.Millisecond)
		cnt++
	}
}

// Apply线程
//
// 把已提交的日志条目应用到状态机
func (rf *Raft) ApplyGoroutine() {
	rf.mu.Lock()

	for !rf.killed() {
		if rf.CommitIndex > rf.LastApplied {
			rf.LastApplied++
			msg := ApplyMsg{
				Command:       rf.Log[rf.V2PIndex(rf.LastApplied)].Command,
				CommandIndex:  rf.LastApplied,
				CommandValid:  true,
				SnapshotValid: false,
			}

			if DEBUG_Aped {
				DPrintf("[APLY] p%d(%v,%d) \"CMIT(%d) APLY(%d+1) apply %v\"\n",
					rf.me, STATE[rf.State], rf.CurrentTerm, rf.CommitIndex, rf.LastApplied-1,
					rf.Log[rf.V2PIndex(msg.CommandIndex)])
			}

			rf.mu.Unlock()
			rf.ApplyCh <- msg
			rf.mu.Lock()
		} else {
			rf.ApplyCond.Wait()
		}
	}
}

// 创建服务器
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{
		mu:        sync.Mutex{},
		peers:     peers,
		persister: persister,
		me:        me,

		VotedFor:    -1,
		dead:        0,
		CurrentTerm: 0,
		CommitIndex: 0,
		LastApplied: 0,

		// MatchIndex和NextIndex应该当选leader再初始化
		MatchIndex: make([]int, 0),
		NextIndex:  make([]int, 0),
		Log:        make([]Log, 0),

		State:     FOLLOWER,
		ApplyCh:   applyCh,
		LogLength: 0,

		SnapshotData:      nil,
		LastIncludedIndex: 0,
		LastIncludedTerm:  0,
	}

	rf.ApplyCond = sync.NewCond(&rf.mu)

	rf.Log = append(rf.Log, Log{0, 0})
	rf.LogLength = 1

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState(), persister.ReadSnapshot())

	// DEBUG
	DPrintf("[MAKE] p%d(%v,%d) \"Log[0:%d:%d) CMIT:%d APLY:%d\"\n",
		rf.me, STATE[rf.State], rf.CurrentTerm,
		rf.LastIncludedIndex, rf.LogLength,
		rf.CommitIndex, rf.LastApplied)

	// 心跳线程
	go rf.HeartBeatGoroutine(50)
	// 选举线程
	go rf.ElectionGoroutine()
	// apply线程
	go rf.ApplyGoroutine()

	return rf
}
