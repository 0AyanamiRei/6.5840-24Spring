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

	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
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
	DEBUG_Make      bool = false
	DEBUG_ticker    bool = false
	DEBUG_Vote      bool = true
	DEBUG_Heartbeat bool = false
	DEBUG_VoteRpc   bool = true
	DEBUG_HeartRpc  bool = false
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
	Log         []Log // 存放日志
	CurrentTerm int   // 任期
	VotedFor    int   // 已投candidate的ID
	VotedTerm   int   // 投票的任期
	CommitIndex int   // 已提交日志的index
	LastApplied int   // 最新的已写日志条目的index
	NextIndex   []int // 发送给其他服务器的日志index初始化为LastApplied
	MatchIndex  []int // 其他服务器日志中与leader日志最后一个相同的index
	// 自定义状态
	LastHearbeats time.Time // 接收心跳消息的时间
	State         int       //确认是否为领导人
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
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
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

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.CurrentTerm
	reply.VoteGranted = false

	var logTerm int
	if len(rf.Log) > 0 && rf.LastApplied < len(rf.Log) {
		logTerm = rf.Log[rf.LastApplied].LogTerm
	} else {
		logTerm = 0
	}

	if rf.VotedTerm < args.Term &&
		args.LastLogIndex >= rf.LastApplied &&
		args.LastLogTerm >= logTerm {
		reply.VoteGranted = true
		rf.VotedTerm = args.Term
		rf.VotedFor = args.CandidateId
		rf.CurrentTerm = args.Term
	}

	if DEBUG_Vote {
		if reply.VoteGranted {
			DPrintf("[Vote(%d)] p%d(%s)-%d: Voted p%d->p%d\n",
				args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, rf.me, rf.VotedFor)
		} else {
			DPrintf("[Vote(%d)] p%d(%s)-%d: Refuse, had vote p%d->p%d\n",
				args.Term, rf.me, STATE[rf.State], rf.CurrentTerm, rf.me, rf.VotedFor)
		}
	}
}

// 调用前已获得 rf.mu
func (rf *Raft) sendRequestVote() {
	rf.CurrentTerm++
	rf.VotedFor = rf.me
	rf.VotedTerm = rf.CurrentTerm
	vote := 1
	var mu sync.Mutex

	args := RequestVoteArgs{
		Term:         rf.CurrentTerm,
		CandidateId:  rf.me,
		LastLogIndex: rf.LastApplied,
	}

	if len(rf.Log) > 0 && rf.LastApplied < len(rf.Log) {
		args.LastLogTerm = rf.Log[rf.LastApplied].LogTerm
	} else {
		args.LastLogTerm = 0
	}

	if DEBUG_Vote {
		DPrintf("[Vote(%d)] p%d(%s)-%d: Vote Start\n",
			rf.CurrentTerm, rf.me, STATE[rf.State], rf.CurrentTerm)
	}

	// 向其他服务器发送投票请求
	for i := range rf.peers {
		if i != rf.me {
			go func(i int) {
				reply := &RequestVoteReply{}
				if DEBUG_VoteRpc {
					DPrintf("[rpc] Vote sent  p%d->p%d\n", rf.me, i)
				}
				if !rf.peers[i].Call("Raft.RequestVote", &args, reply) {
					if DEBUG_VoteRpc {
						DPrintf("[rpc] Vote reply p%d->p%d failed\n", rf.me, i)
					}
					return
				} else {
					if DEBUG_VoteRpc {
						DPrintf("[rpc] Vote reply p%d->p%d succeed\n", rf.me, i)
					}
				}
				mu.Lock() // 这把锁除了保护vote, 还可以用来保护rf的内容
				defer mu.Unlock()

				if reply.VoteGranted {
					vote++
					if vote > len(rf.peers)/2 {

						if DEBUG_Vote {
							DPrintf("[Vote(%d)] p%d(%s)-%d: Vote enough with %dvotes\n",
								rf.CurrentTerm, rf.me, STATE[rf.State], rf.CurrentTerm, vote)
						}

						rf.State = LEADER
						rf.sendHeartbeat()
						rf.LastHearbeats = time.Now()
					}
				} else if reply.Term > rf.CurrentTerm {
					rf.CurrentTerm = reply.Term
				}

			}(i)
		}
	}
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
}

// 调用前已获得 rf.mu
func (rf *Raft) sendHeartbeat() {
	term := rf.CurrentTerm
	me := rf.me
	var mu sync.Mutex

	args := AppendEntriesArgs{
		Term:     term,
		LeaderId: me,
	}
	// 更新自己
	rf.LastHearbeats = time.Now()

	if DEBUG_Heartbeat {
		DPrintf("[Heart] p%d(%s)-%d: !!!Heartbeat!!!\n",
			rf.me, STATE[rf.State], rf.CurrentTerm)
	}

	for i := range rf.peers {
		if i != me && rf.State == LEADER {
			go func(i int) {
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
				// leader收到心跳消息的回复发现自己的任期过时
				if rf.CurrentTerm < reply.Term {
					mu.Lock()
					defer mu.Unlock()

					rf.CurrentTerm = reply.Term
					rf.State = FOLLOWER
				}
			}(i)
		}
	}
}

// 接收心跳消息
func (rf *Raft) Heartbeat(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.CurrentTerm
	if args.Term >= rf.CurrentTerm {
		rf.LastHearbeats = time.Now()
		rf.CurrentTerm = args.Term
		rf.State = FOLLOWER
	}

	if DEBUG_Heartbeat {
		DPrintf("[Heart] p%d(%s)-%d: p%d->p%d beat!!!\n",
			rf.me, STATE[rf.State], rf.CurrentTerm, args.LeaderId, rf.me)
	}
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.

func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	State := true

	// Your code here (3B).

	return index, term, State
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
		ms := 500 + (rand.Int63() % 400)

		time.Sleep(time.Duration(ms) * time.Millisecond)

		rf.mu.Lock()

		if rf.State == FOLLOWER && time.Since(rf.LastHearbeats) > time.Duration(ms)*time.Millisecond {
			// rf.State = CANDIDATE, 暂时不设置, 不知道有什么用
			// 调用sendRequestVote需持有rf.mu
			rf.sendRequestVote()
		}

		rf.mu.Unlock()
	}
}

// 心跳线程
func (rf *Raft) heartbeat(heartTime int) {
	for !rf.killed() {
		rf.mu.Lock()
		if rf.State == LEADER {
			// leader发送消息
			rf.sendHeartbeat()
		}
		rf.mu.Unlock()
		time.Sleep(time.Duration(heartTime) * time.Microsecond)
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
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

		MatchIndex: make([]int, 0),
		NextIndex:  make([]int, 0),
		Log:        make([]Log, 0),

		State: FOLLOWER,
	}

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// 心跳线程
	go rf.heartbeat(100000)
	// 选举线程
	go rf.ticker()

	return rf
}
