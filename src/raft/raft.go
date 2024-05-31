package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//

// rf = Make(...)
//   create a new Raft server.

// rf.Start(command interface{}) (index, term, State)
//   start agreement on a new log entry

// rf.GetState() (term, State)
//   ask a Raft for its current term, and whether it thinks it is leader

// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.

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
	DEBUG           bool = true
	DEBUG_Info      bool = true
	DEBUG_Vote      bool = true
	DEBUG_Heartbeat bool = false
	DEBUG_Aped      bool = true
	DEBUG_VoteRpc   bool = false
	DEBUG_HeartRpc  bool = false
	DEBUG_ApedRpc   bool = false
	DEBUG_Persist   bool = false
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
	VotedTerm   int   // 投票的任期
	CommitIndex int   // 已提交的日志(大多数服务器写入Log后就算作已提交)
	LastApplied int   // 已执行的日志(已提交的日志分为已执行和未执行)
	NextIndex   []int // 发送到该服务器的下一个日志条目的索引
	MatchIndex  []int // 已知的已经复制到该服务器的最高日志条目的索引
	// 自定义状态
	LastHearbeats time.Time // 接收心跳消息的时间
	State         int       //确认是否为领导人
	// 提供给应用的接口
	ApplyCh chan ApplyMsg
	HeartCh chan struct{}
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
	CommandIndex int

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
	LastLogIndex int
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
	PrevLogIndex int   // 简单的一致性检查
	PrevLogTerm  int   // 简单的一致性检查
	Entries      []Log // 日志内容(记得附加此前任期的日志内容)
	LeaderCommit int   // leader追踪的已提交日志下标
}

type AppendEntriesReply struct {
	Term    int  // follower的任期号, 以便leader更新
	Success bool // 日志添加是否成功
	// fast bakeup
	XTerm  int // 冲突的任期号, -1表示该日志条目不存在
	XLen   int // 如果XTerm=-1, 记录leader应该追加日志的起始下标
	XIndex int // 如果XTerm!=-1, 记录该任期第一个日志条目的下标
}

func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.CurrentTerm, rf.State == LEADER
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
//
// 需要做持久化的状态: (1)CurrentTerm, (2)VotedFor(VotedTerm), (3)Log
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.CurrentTerm)
	e.Encode(rf.VotedFor)
	e.Encode(rf.VotedTerm)
	e.Encode(rf.Log)
	raftstate := w.Bytes()
	rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var CurrentTerm int
	var VotedFor int
	var VotedTerm int
	var Log []Log
	if d.Decode(&CurrentTerm) != nil || d.Decode(&VotedFor) != nil ||
		d.Decode(&VotedTerm) != nil || d.Decode(&Log) != nil {
		if DEBUG_Persist {
			DPrintf("[Persist] p%d readPersist failed\n", rf.me)
		}
		return
	} else {
		rf.CurrentTerm = CurrentTerm
		rf.VotedFor = VotedFor
		rf.VotedTerm = VotedTerm
		rf.Log = Log
	}
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).
}

//************************************************************************************
// RPCs
//************************************************************************************

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
		if rf.State == LEADER {
			DPrintf("leader p%d(%v,%d) vote!!!\n", rf.me, STATE[rf.State], rf.CurrentTerm)
			rf.State = FOLLOWER
		}
		rf.CurrentTerm = args.Term
	}

	//1: 任期过时
	if rf.CurrentTerm > args.Term {
		reason |= 1 << 1
	}

	//2: 已投票
	if rf.VotedTerm == args.Term {
		reason |= 1 << 2
	}

	// 较新的日志: (最后一个条目任期更大) || (最后一个条目任期相同&&下标更大)
	//3: 候选人日志不如自己新
	if !((args.LastLogTerm > rf.Log[len(rf.Log)-1].LogTerm) ||
		(args.LastLogTerm == rf.Log[len(rf.Log)-1].LogTerm &&
			args.LastLogIndex >= len(rf.Log)-1)) {
		reason |= 1 << 3
	}

	//4: 投票成功
	if reason == 0 {
		reply.VoteGranted = true
		rf.CurrentTerm = args.Term
		rf.LastHearbeats = time.Now()
		rf.VotedFor = args.CandidateId
		rf.VotedTerm = args.Term
	}

	// 持久化
	if reason == 0 || rf.CurrentTerm < args.Term {
		rf.persist()
	}

	if DEBUG_Vote {
		if reason == 0 {
			DPrintf("[Vote-%d] p%d(%v,%d)->p%d \"OKKKK\"\n",
				args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, args.CandidateId)
		} else {
			DPrintf("[Vote-%d] p%d(%v,%d)->p%d \"Refuse Vote:(%d)\"\n",
				args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, args.CandidateId, reason)
		}
	}
}

// 候选人发起本轮投票
// 1: 首先给自己投票并自增任期号
// 2: 当发起投票时, 重置你的选举超时定时器
// 3: 成为领导人后需要做一些初始工作
//
// 调用前未获得 rf.mu
func (rf *Raft) sendRequestVote() {
	rf.mu.Lock()
	// 1:
	rf.CurrentTerm++
	rf.VotedFor = rf.me
	rf.VotedTerm = rf.CurrentTerm
	rf.persist() // 持久化
	Len := len(rf.Log)

	var vote uint32 = 1

	args := RequestVoteArgs{
		Term:         rf.CurrentTerm,
		CandidateId:  rf.me,
		LastLogIndex: Len - 1,
		LastLogTerm:  rf.Log[Len-1].LogTerm,
	}
	// 2:
	rf.LastHearbeats = time.Now()

	if DEBUG_Vote {
		DPrintf("[Vote-%d] p%d(%v, %d): Vote Start\n",
			args.Term, rf.me, STATE[rf.State], rf.CurrentTerm)
	}

	rf.mu.Unlock()

	// 向其他服务器发送投票请求
	for i := range rf.peers {
		if i != rf.me {
			go func(i int) {

				reply := &RequestVoteReply{}

				if DEBUG_Vote {
					DPrintf("[Vote-%d] p%d(%v,%d)->p%d \"Ask Vote\"\n",
						args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, i)
				}

				if !rf.peers[i].Call("Raft.RequestVote", &args, reply) {
					if DEBUG_VoteRpc {
						DPrintf("[rpc] Vote reply p%d->p%d failed\n", rf.me, i)
					}
				} else {
					if DEBUG_VoteRpc {
						DPrintf("[rpc] Vote reply p%d->p%d succeed\n", rf.me, i)
					}
				}

				rf.mu.Lock()
				// 处理过期的rpc回复
				if rf.CurrentTerm != args.Term {
					rf.mu.Unlock()
					return
				}
				rf.mu.Unlock()

				if reply.VoteGranted {
					// 3: 发一轮心跳消息, 并且初始化NextIndex和MatchIndex
					if v := atomic.AddUint32(&vote, 1); v > (uint32(len(rf.peers) / 2)) {
						if DEBUG_Vote {
							DPrintf("[Vote-%d] p%d(%v,%d): Got %d votes\n",
								args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, vote)
						}

						rf.mu.Lock()
						// 当选后重新初始化
						LogLen := len(rf.Log)
						for i := range rf.peers {
							if i >= len(rf.NextIndex) {
								rf.NextIndex = append(rf.NextIndex, LogLen)
							} else {
								rf.NextIndex[i] = LogLen
							}
							if i >= len(rf.MatchIndex) {
								rf.MatchIndex = append(rf.MatchIndex, 0)
							} else {
								rf.MatchIndex[i] = 0
							}
						}
						rf.State = LEADER
						rf.LastHearbeats = time.Now()
						rf.mu.Unlock()
					}
				}

				rf.mu.Lock()
				// 自己任期过期
				if !reply.VoteGranted && reply.Term > rf.CurrentTerm {
					rf.CurrentTerm = reply.Term
					rf.persist() // 持久化
					rf.State = FOLLOWER
				}
				rf.mu.Unlock()
			}(i)
		}
	}
}

// 或许在心跳消息里附带日志比较不错
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
}

// 调用前未获得 rf.mu
func (rf *Raft) sendHeartbeat() {
	rf.mu.Lock()
	// 更新自己
	rf.LastHearbeats = time.Now()
	// 提前复制资源, 以释放锁
	term := rf.CurrentTerm
	commitIndex := rf.CommitIndex
	prevIndex := make([]int, len(rf.peers))
	prevTerm := make([]int, len(rf.peers))
	entries := make([][]Log, len(rf.peers))
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		prevIndex[i] = rf.NextIndex[i] - 1
		prevTerm[i] = rf.Log[rf.NextIndex[i]-1].LogTerm
		// 根据日志长度和nextIndex来决定是否附加日志
		if len(rf.Log)-1 >= rf.NextIndex[i] {
			entries[i] = make([]Log, len(rf.Log[rf.NextIndex[i]:]))
			copy(entries[i], rf.Log[rf.NextIndex[i]:])

			if DEBUG_Aped {
				DPrintf("[APED] p%d(%v,%d)->p%d \"Aped Log[%d~] %v\"\n",
					rf.me, STATE[rf.State], rf.CurrentTerm, i, rf.NextIndex[i], entries[i])
			}

		} else {
			entries[i] = make([]Log, 0)
			if DEBUG_Heartbeat {
				DPrintf("[Heart(%d)] sent p%d(%s)->p%d\n", rf.CurrentTerm, rf.me, STATE[rf.State], i)
			}
		}
	}
	if DEBUG_Heartbeat {
		DPrintf("[Heart] p%d(%s)-%d: !!!Heartbeat!!!\n",
			rf.me, STATE[rf.State], rf.CurrentTerm)
	}
	rf.mu.Unlock()

	// 并发发送心跳&追加日志消息
	for i := range rf.peers {
		if i != rf.me {
			// 并行发送
			go func(i int) {
				//在这些并行的线程中保护变量
				args := AppendEntriesArgs{
					Term:         term,
					LeaderId:     rf.me,
					PrevLogIndex: prevIndex[i],
					PrevLogTerm:  prevTerm[i],
					LeaderCommit: commitIndex,
					Entries:      entries[i],
				}

				reply := AppendEntriesReply{}

				if DEBUG_HeartRpc {
					DPrintf("[rpc] Heart sent  p%d->p%d\n", rf.me, i)
				}
				if !rf.peers[i].Call("Raft.Heartbeat", &args, &reply) {
					if DEBUG_HeartRpc {
						DPrintf("[rpc] Heart reply p%d->p%d failed\n", i, rf.me)
					}
					return
				} else {
					if DEBUG_HeartRpc {
						DPrintf("[rpc] Heart reply p%d->p%d succeed\n", i, rf.me)
					}
				}

				rf.mu.Lock()
				defer rf.mu.Unlock()

				// rpc的回复过期,直接放弃
				if rf.CurrentTerm != args.Term {
					return
				}

				// leader的任期过时导致失败
				if args.Term < reply.Term {
					if rf.CurrentTerm < reply.Term {
						rf.CurrentTerm = reply.Term
						rf.persist() // 持久化
					}
					rf.State = FOLLOWER
					return
				}

				// 一致性检查+任期检查通过
				if reply.Success {
					// 助教提示如果设置nextIndex[i]-1或len(log)不安全
					rf.MatchIndex[i] = args.PrevLogIndex + len(args.Entries)
					rf.NextIndex[i] = rf.MatchIndex[i] + 1

					// 此次为附加日志的心跳消息, 根据Rules for Servers中Leader的第四条要求
					// 更新rf.CommitIndex = N(大多数服务器持有日志的下标)
					if len(args.Entries) != 0 {
						N := len(rf.Log) - 1
						for N > rf.CommitIndex {
							count := 0 // 统计"大多数"
							for j := range rf.MatchIndex {
								if j == rf.me {
									count++
								} else if rf.MatchIndex[j] >= N && rf.Log[N].LogTerm == rf.CurrentTerm {
									count++
								}
							}
							// 更新rf.CommitIndex
							if count > len(rf.peers)/2 {
								rf.CommitIndex = N
								break
							}
							N--
						}
					}
					return
				}

				// 日志任期冲突导致失败(这里采取了fast backup优化, 正常来说让rf.NextIndex[i]--即可)
				if reply.XTerm != -1 {
					for j := range rf.Log {
						// 找到该冲突任期在leader日志中的下标
						if rf.Log[j].LogTerm == reply.XTerm {
							rf.NextIndex[i] = j + 1
							return
						}
					}
					// leader日志中不存在该冲入任期
					rf.NextIndex[i] = reply.XIndex
					return
				} else { // 检查日志不存在
					rf.NextIndex[i] = reply.XLen
					return
				}
			}(i)
		}
	}
}

// 接收心跳消息
// 0: rpc共通检查, 如果自己的任期落后, 更新接收方的任期号, 并且更新状态为FOLLOWER
// 1(任期落后): 如果领导人的任期比自己小, 那么放弃这次心跳消息, 不重置选举计时器
// 2(一致性检查):
// -2a) 被检查日志条目不存在
// -2b) 被检查日志条目任期冲突
func (rf *Raft) Heartbeat(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	Len := len(rf.Log)
	reply.Term = rf.CurrentTerm

	// 0
	if args.Term > rf.CurrentTerm {
		rf.CurrentTerm = args.Term
		rf.State = FOLLOWER
		rf.persist() // 持久化
	}

	// 1:领导人任期落后
	if args.Term < rf.CurrentTerm {
		if DEBUG_Aped {
			DPrintf("[HERT] p%d(%v,%d) \"p%d(L,%d) Out of data\"\n",
				rf.me, STATE[rf.State], rf.CurrentTerm, args.LeaderId, args.Term)
		}
		reply.Success = false
		return
	}

	//(除了rpc任期过期外都需要选举计时器)
	rf.LastHearbeats = time.Now()

	// 2a: 被检查日志条目不存在
	if args.PrevLogIndex > len(rf.Log)-1 {
		reply.XTerm = -1
		reply.XLen = len(rf.Log)
		reply.Success = false
		return
	}
	// 2b: 被检查日志条目任期冲突
	if args.PrevLogTerm != rf.Log[args.PrevLogIndex].LogTerm {
		reply.XTerm = rf.Log[args.PrevLogIndex].LogTerm
		// 找到该任期在日志第一次出现的下标
		for i := range rf.Log {
			if rf.Log[i].LogTerm == reply.XTerm {
				reply.XIndex = i
			}
		}
		reply.Success = false
		return
	}

	// 附加了日志(这里通过了一致性检查+任期检查)
	// 从rf.Log[args.PrevLogIndex + 1]开始覆盖或追加日志内容
	if len(args.Entries) != 0 {
		index := args.PrevLogIndex + 1

		if DEBUG_Aped {
			DPrintf("[APED] p%d(%v,%d) \"CMIT(%d) APLY(%d) Add Log[%d~] %v\"\n",
				rf.me, STATE[rf.State], rf.CurrentTerm, rf.CommitIndex, rf.LastApplied, index, args.Entries)
		}

		if Len > index {
			rf.Log = append(rf.Log[:index], args.Entries...)
		} else {
			rf.Log = append(rf.Log, args.Entries...)
		}

		rf.persist() // 持久化
	}

	reply.Success = true

	// 针对重连上来的leader
	rf.State = FOLLOWER

	// AppendEntries RPC的第五条建议
	if args.LeaderCommit > rf.CommitIndex {
		cmit := rf.CommitIndex
		if args.LeaderCommit < len(rf.Log)-1 {
			rf.CommitIndex = args.LeaderCommit
		} else {
			rf.CommitIndex = len(rf.Log) - 1
		}
		if DEBUG_Aped {
			DPrintf("[CMIT) p%d(%v,%d) \"CMIT(%d+%d) APLY(%d) Commit Log[%d~] %v\"\n",
				rf.me, STATE[rf.State], rf.CurrentTerm, cmit, rf.CommitIndex-cmit, rf.LastApplied,
				cmit, rf.Log[cmit:rf.CommitIndex+1])
		}
	}

	if DEBUG_Heartbeat {
		DPrintf("[Heart] p%d(%s)-%d: p%d->p%d beat!!!\n",
			rf.me, STATE[rf.State], rf.CurrentTerm, args.LeaderId, rf.me)
	}
}

// 客户端发出的指令则调用Start, 返回index, CurrentTerm,isLeader
// Q: Start函数需要等待日志提交后才返回吗
// A: Start函数不保证command一定提交, 不需要等待, 只是追加到leader的log中
//
// 1: 追加到leadr自己的Log中
// 2: 向其他服务器发送AppendEntries消息
// (暂且如此, 有的建议是附加在心跳消息中, 可以在负载很大的时候减少rpcs)
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.State != LEADER {
		return -1, -1, false
	}
	// 1:
	entry := Log{LogTerm: rf.CurrentTerm, Command: command}
	rf.Log = append(rf.Log, entry)
	rf.persist() // 持久化

	if DEBUG_Aped {
		DPrintf("[Start] p%d(%v,%d) \"Add Log:%v\"\n", rf.me, STATE[rf.State], rf.CurrentTerm, entry)
	}

	// 2: (伴随心跳消息一起发送了)
	// leader追加日志后会使len(rf.Log) > rf.NextIndex[i]
	//rf.sendHeartbeat()
	//rf.HeartCh <- struct{}{}

	return len(rf.Log) - 1, rf.CurrentTerm, true
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	//DPrintf("p%d was killed\n", rf.me)
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// 选举线程
func (rf *Raft) ticker() {
	for !rf.killed() {
		ms := 350 + (rand.Int63() % 200)
		time.Sleep(time.Duration(ms) * time.Millisecond)

		rf.mu.Lock()
		if rf.State == FOLLOWER && time.Since(rf.LastHearbeats) > time.Duration(ms)*time.Millisecond {
			rf.mu.Unlock()
			rf.sendRequestVote()
		} else {
			rf.mu.Unlock()
		}
	}
}

// 心跳线程
func (rf *Raft) heartbeat(heartTime int) {
	cnt := 0
	for !rf.killed() {
		rf.mu.Lock()
		if cnt%5 == 0 && DEBUG_Info {
			DPrintf("[Info] p%d(%v,%d) \"CMIT(%d) APLY(%d) Log:%v\"",
				rf.me, STATE[rf.State], rf.CurrentTerm, rf.CommitIndex, rf.LastApplied, rf.Log)
		}
		if rf.State == LEADER {
			rf.mu.Unlock()
			rf.sendHeartbeat()
		} else {
			rf.mu.Unlock()
		}

		time.Sleep(time.Duration(heartTime) * time.Millisecond)
		cnt++
	}
}

// applier线程
func (rf *Raft) applier() {

	buffer := ApplyMsg{
		CommandValid: false,
	}

	for !rf.killed() {
		time.Sleep(10 * time.Millisecond)

		rf.mu.Lock()
		if rf.CommitIndex > rf.LastApplied {
			if DEBUG_Aped {
				DPrintf("[APLY] p%d(%v,%d) \"CMIT(%d) APLY(%d+1) apply Log[%d]: %v\"\n",
					rf.me, STATE[rf.State], rf.CurrentTerm, rf.CommitIndex, rf.LastApplied, rf.LastApplied+1,
					rf.Log[rf.LastApplied+1])
			}
			buffer.Command = rf.Log[rf.LastApplied+1].Command
			buffer.CommandIndex = rf.LastApplied + 1
			buffer.CommandValid = true
			rf.LastApplied++
		}
		rf.mu.Unlock()

		if buffer.CommandValid {
			rf.ApplyCh <- buffer
			buffer.CommandValid = false
		}
	}
}

func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{
		mu:        sync.Mutex{},
		peers:     peers,
		persister: persister,
		me:        me,

		VotedFor:    -1,
		VotedTerm:   0,
		dead:        0,
		CurrentTerm: 0,
		CommitIndex: 0,
		LastApplied: 0,

		// MatchIndex和NextIndex应该当选leader再初始化
		MatchIndex: make([]int, 0),
		NextIndex:  make([]int, 0),
		Log:        make([]Log, 0),

		State:   FOLLOWER,
		ApplyCh: applyCh,
	}

	// 不知道为什么测试的下标从1开始, 那第一个就刚好拿来做初始化
	rf.Log = append(rf.Log, Log{0, 0})

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// 心跳线程
	go rf.heartbeat(125)
	// 选举线程
	go rf.ticker()
	// apply线程
	go rf.applier()

	return rf
}
