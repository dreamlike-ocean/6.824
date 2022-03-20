package raft

import (
	"6.824/labrpc"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
// 这是raft必须向service或者test必须暴露的api。请查看下面每一个方法上的注释来了解更多信息
// rf = Make(...)
//   创建一个raft服务器
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry 开始就新的日志条目达成协议
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
//    询问 Raft 的当前任期，以及它是否认为自己是领导者
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//  每次向日志提交新条目时，每个 Raft 对等点都应向同一服务器中的服务（或测试者）发送 ApplyMsg。
//

// ApplyMsg
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}
type AppendMsg struct {
	Term     int
	LeaderId int

	//todo
}

const HeartBeatInterval = 120
const ElectronInterval = 300
const (
	follower = iota
	candidate
	lead
)

// Raft
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	lastHeartBeat time.Time //2A
	leadIndex     int
	term          int
	voteFor       int
	status        int
}

// GetState
// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	var term = rf.term
	var isleader = rf.status == lead
	// Your code here (2A).
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
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

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int //  候选者任期
	CandidateId  int //候选者编号
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}
type AppendReply struct {
	Term    int
	Success bool
}

// RequestVote
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.lastHeartBeat = time.Now()
	if rf.term > args.Term {
		reply.Term = rf.term
		reply.VoteGranted = false
		return
	}

	if rf.term < args.Term {
		rf.becomeFollower(args.Term)
	}

	if rf.voteFor == -1 || rf.voteFor == args.CandidateId {
		rf.voteFor = args.CandidateId
		//reply.Term = rf.term
		reply.VoteGranted = true
	}
	return
}

func (rf *Raft) AppendRequest(msg *AppendMsg, reply *AppendReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.lastHeartBeat = time.Now()
	if msg.Term < rf.term {
		reply.Term = rf.term
		reply.Success = false
		return
	}
	//此时心跳term大于自己的term
	if rf.status != follower {
		rf.becomeFollower(msg.Term)
	}
	rf.term = msg.Term
	DPrintf("%d get heartbeat from %d,term %d", rf.me, msg.LeaderId, msg.Term)
	rf.leadIndex = msg.LeaderId
	reply.Success = true
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
// 一个模拟真实有损网络的RPC handler
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}
func (rf *Raft) sendAppendMsg(server int, args *AppendMsg, reply *AppendReply) bool {
	return rf.peers[server].Call("Raft.AppendRequest", args, reply)
}

// Start
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
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

// Kill
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for !rf.killed() {
		timeout := ElectronInterval + rand.Intn(300)
		start := time.Now()
		time.Sleep(time.Millisecond * time.Duration(timeout))
		rf.mu.Lock()
		if rf.lastHeartBeat.Before(start) && rf.status != lead {
			go func() {
				rf.startElectron()
			}()
		}
		rf.mu.Unlock()
	}
}
func (rf *Raft) becomeFollower(term int) {
	rf.status = follower
	rf.term = term
	rf.voteFor = -1
	rf.lastHeartBeat = time.Now()
}
func (rf *Raft) becomeCandidate() {
	rf.term++
	rf.voteFor = rf.me
	rf.lastHeartBeat = time.Now()
	rf.status = candidate
}
func (rf *Raft) becomeLead() {
	rf.status = lead
	rf.lastHeartBeat = time.Now()
}
func (rf *Raft) broadcastHearBeat() {
	for {
		rf.mu.Lock()
		if rf.status != lead || rf.killed() {
			rf.mu.Unlock()
			return
		}
		rf.mu.Unlock()
		for index := range rf.peers {
			if index == rf.me {
				continue
			}
			go func(i int) {
				args := AppendMsg{Term: rf.term, LeaderId: rf.me}
				reply := AppendReply{}
				netSuccess := rf.sendAppendMsg(i, &args, &reply)
				if !netSuccess {
					return
				}
				rf.mu.Lock()
				defer rf.mu.Unlock()
				//发现新的leader
				if !reply.Success && reply.Term > rf.term {
					rf.becomeFollower(reply.Term)
				}
			}(index)
		}
		time.Sleep(HeartBeatInterval * time.Millisecond)
	}
}

func (rf *Raft) startElectron() {
	rf.mu.Lock()
	rf.becomeCandidate()
	// 不要在rpc时持有锁 防止和接收心跳并发时 心跳那边卡住
	rf.mu.Unlock()
	args := RequestVoteArgs{Term: rf.term, CandidateId: rf.me}
	DPrintf("%d wake up to startElector,term:%d", rf.me, rf.term)
	voteLock := sync.Mutex{}
	voteCount := 1
	// 用waitGroup设计的不好 换成条件变量试试
	condition := sync.NewCond(&voteLock)
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go func(peerId int) {
			reply := RequestVoteReply{}
			netSuccess := rf.sendRequestVote(peerId, &args, &reply)
			if netSuccess && reply.VoteGranted {
				DPrintf("%d vote to %d", peerId, rf.me)
				voteLock.Lock()
				voteCount++
				if voteCount > len(rf.peers)/2 {
					condition.Signal()
				}
				voteLock.Unlock()
			}
		}(i)
	}
	voteLock.Lock()
	condition.Wait()
	voteLock.Unlock()

	if rf.status == candidate {
		DPrintf("%d win", rf.me)
		rf.mu.Lock()
		rf.becomeLead()
		rf.mu.Unlock()

		rf.broadcastHearBeat()
	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.voteFor = -1
	rf.status = follower
	// Your initialization code here (2A, 2B, 2C).
	rf.lastHeartBeat = time.Now()
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
