package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
	"labrpc"
)
// import "bytes"
// import "labgob"
//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//

const (
	Follower  = "FOLLOWER"
	Candidate = "CANDIDATE"
	Leader    = "LEADER"
)

type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

type LogEntry struct {
	Command interface{}
	LastLogTerm int
	LastLogIndex int
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	//candidate’s term
	Term int
	//candidate requesting vote
	CandidateId int
	//index of candidate’s last log entry
	LastLogIndex int
	//term of candidate’s last log entry
	LastLogTerm int

}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term int  //currentTerm, for candidate to update itself
	VoteGranted bool //true means candidate received vote

}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	///
	currentState string
	//log[] LogEntry
	/////////Volatile state on all servers//////////
	commitIndex int
	lastApplied int
	///////// Volatile state on leaders /////////
	nextIndex[] int //initialized to leader  last log index + 1)
	matchIndex[] int //index of highest log entry  known to be replicated on server. Initialized to 0, increases monotonically

	/////////Persistent state on all servers //////////
	///Latest term server has seen
	currentTerm int
	///CandidateId that received vote in current term
	votedFor int
	/// Place holder for log entries.
	log         []LogEntry
	electionTimer *time.Timer

	applyCh chan ApplyMsg
}


// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).
	term = rf.currentTerm
	if rf.currentState =="LEADER"{
		isleader =true
	} else{
		isleader =false
	}
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
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	rf.mu.Lock()
	rf.debug("***************Inside the RPC handler for sendRequestVote *********************")
	defer rf.mu.Unlock()
	var lastIndex int
	//var lastTerm  int
	if len(rf.log) > 0 {
		lastLogEntry := rf.log[len(rf.log)-1]
		lastIndex = lastLogEntry.LastLogIndex
		//lastTerm = lastLogEntry.lastLogTerm
	}else{
		lastIndex = 0
		//lastTerm = 0
	}
	reply.Term = rf.currentTerm
	//rf.debug()
	if args.Term < rf.currentTerm {
		reply.VoteGranted = false
		rf.debug("My term is higher than candidate's term, myTerm = %d, candidate's term = %d", rf.currentTerm,args.Term )
	} else if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && args.LastLogIndex >= lastIndex {
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
		rf.currentTerm = args.Term
		rf.resetElectionTimer()
		//rf.debug("I am setting my currentTerm to -->",args.Term,"I am ",rf.me)
	}
}

// This is the receiver for append entries.

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply)  {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// Resetting as we received a heart beat.
	rf.resetElectionTimer()
	rf.debug( "AppendEntries: from LEADER %#v \n",args)
	rf.debug("My current state: %#v \n", rf)
	//1. Reply false if term < currentTerm (§5.1)
	if args.Term > rf.currentTerm{
		if rf.currentState != Follower {
			rf.transitionToFollower(args.Term)
		}
	}
	//2. Reply false if log doesn’t contain an entry at prevLogIndex
	//whose term matches prevLogTerm (§5.3)
	//3. If an existing entry conflicts with a new one (same index
	//but different terms), delete the existing entry and all that
	//follow it (§5.3)
	//4. Append any new entries not already in the log
	//5. If leaderCommit > commitIndex, set commitIndex =
	//	min(leaderCommit, index of last new entry)
	/////////////Pending implementation point 5 above.
	if args.Term < rf.currentTerm{
		reply.Success = false
		reply.Term =rf.currentTerm
		return
	}

	// Update my term to that of the leaders
	rf.currentTerm = args.Term
	rf.debug("Dereferencing %d",len(rf.log)-1)
	rf.debug("Current log contents %v", rf.log)

	// Check first whether it is a heartbeat or an actual append entry.
	// If it is heartbeat, then just reset the timer and then go back.
	//Otherwise, we need to add the entries into the logs of this peer.
	// If this is heart beat, then we know that the command is going to be nil.
	// Identify this and return.
	lastLogEntryIndex := len(rf.log) - 1
	if args.LogEntries ==  nil {
		//This is heart beat
		reply.Term = rf.currentTerm
		rf.debug("Received a HEART BEAT.")
	}else {
		rf.debug("Received an APPEND ENTRY. PROCESSING")
		lastLogEntry := rf.log[len(rf.log)-1]
		//1a
		if lastLogEntryIndex < args.PreviousLogIndex {
			reply.Success = false
			reply.NextIndex = lastLogEntryIndex
			rf.debug("1a \n")
			return
		}
		//1b
		if lastLogEntryIndex > args.PreviousLogIndex {
			reply.Success = false
			rf.debug("Last log entry index --> %d, PreviousLogIndex From LEADER -->%d", lastLogEntryIndex, args.PreviousLogIndex)
			rf.log = rf.log[:len(rf.log)-1]
			return
		}
		//3
		if lastLogEntry.LastLogTerm != args.PreviousLogTerm {
			reply.Success = false
			//Reduce size by 1;
			rf.debug("3 \n")
			rf.log = rf.log[:len(rf.log)-1]
			return
		}

		// 4 We are good to apply the command.
		rf.printSlice(rf.log, "Before")
		rf.debug("Printing the entry to be added within the handler %v", args.LogEntries)
		rf.log = append(rf.log, args.LogEntries...)
		rf.printSlice(rf.log, "After")
		rf.debug("\n Applied the command to the log. Log size is -->%d \n", len(rf.log))
		//5
	}
	if args.LeaderCommit >rf.commitIndex {
		rf.debug("5 Update commitIndex. LeaderCommit %v  rf.commitIndex %v \n",args.LeaderCommit,rf.commitIndex )
		//Check whether all the entries are committed prior to this.
		oldCommitIndex:=rf.commitIndex
		rf.commitIndex = min(args.LeaderCommit,lastLogEntryIndex+1)
		rf.debug("moving ci from %v to %v", oldCommitIndex, rf.commitIndex)
		//Send all the received entries into the channel
		j:=0
		for i:=oldCommitIndex ;i<args.LeaderCommit;i++ {
			rf.debug("Committing %v ",i)
			applyMsg := ApplyMsg{CommandValid: true, Command: rf.log[i].Command, CommandIndex: i}
			j++
			rf.debug("Sent a response to the end client ")
			rf.debug("applyMsg %v",applyMsg)
			rf.applyCh <- applyMsg
		}
	}
	reply.Success = true
	//Check at the last. This is because this way the first HB will be sent immediately.
	//timer := time.NewTimer(100 * time.Millisecond)
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
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

//
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

	//Check if I am the leader
	//if false  --> return
	//If true -->
	// 1. Add to my log
	// 2. Send heart beat/Append entries to other peers
	//Check your own last log index and 1 to it.
	//Let other peers know that this is the log index for new entry.

	// we need to modify the heart beat mechanism such that it sends entries if any.
	index := -1
	term := -1
	//Otherwise prepare the log entry from the given command.
	// Your code here (2B).
	///////
	term, isLeader :=rf.GetState()
	if isLeader == false {
		return index,term,isLeader
	}
	term = rf.currentTerm
	index = rf.commitIndex
	rf.sendAppendLogEntries(command)
	return index, term, isLeader
}

func (rf* Raft)sendAppendLogEntries(command interface{}){
	if rf.currentState!=Leader{
		return
	}
	var prevLogIndex, prevLogTerm = 0, 0
	if len(rf.log) > 0 {
		lastEntry := rf.log[len(rf.log)-1]
		prevLogIndex, prevLogTerm = lastEntry.LastLogIndex, lastEntry.LastLogIndex
	} else {
		prevLogIndex, prevLogTerm = 0, 0
	}
	logEntry := LogEntry{LastLogIndex: prevLogIndex+1, LastLogTerm: prevLogTerm+1, Command: command}
	// Rule for servers: Leaders:
	//If command received from client: append entry to local log,
	//respond after entry applied to state machine (§5.3)
	rf.log = append(rf.log, logEntry)
	rf.debug("Added the log entry to leader's log ----> %d", len(rf.log))
	//Create a channel to collect all the replies from the peers.
	replyCh := make(chan bool)
	//Otherwise we will have to send append entries for every peer.
	for id, peer := range rf.peers {
		if id != rf.me {
			logEntryArr:= make([]LogEntry, 1)
			logEntryArr[0] = logEntry
			go func(id int, peer *labrpc.ClientEnd) {
				//rf.debug("logEntryArr123 %v", logEntryACurrent log contentsrr[0])
				reply := AppendEntriesReply{}
				args := AppendEntriesArgs{
					Term:             rf.currentTerm,
					LeaderID:         rf.me,
					PreviousLogIndex: prevLogIndex,
					PreviousLogTerm:  prevLogTerm,
					LogEntries:       logEntryArr, //Log Entry array
					LeaderCommit:     rf.commitIndex,
				}
				//rf.debug("logEntryArr123 DEBUG NOW  %v", args)
				requestName := "Raft.AppendEntries"
				ok := rf.peers[id].Call(requestName, &args, &reply)
				//fmt.Println("Called APPEND ENTRIES ***************** ",ok, " ",id)
				//rf.debug("Called APPEND ENTRIES ***************** OK = %d",ok)
				///If everything is ok or not
				// if term>myTerm => Transition to follower.
				if ok && reply.Term>rf.currentTerm {
					rf.mu.Lock()
					rf.transitionToFollower(reply.Term)
					rf.mu.Unlock()
				}
				replyCh <- ok
			}(id,peer)
		}
	}//end of for loop
	count:=0 //Count of total responses
	OKCount:=0 //Count of how many peers have agreed to commit.
	for {
		if rf.currentState!=Leader{
			return
		}
		responseFromPeer:=<-replyCh
		rf.debug("Response from peer %t",responseFromPeer)
		count++
		if responseFromPeer {
			OKCount++
		}

		if OKCount > (len(rf.peers)/2) {
			rf.debug("The peers have agreed to commit")
			applyMsg := ApplyMsg {CommandValid:true,Command:command,CommandIndex:rf.commitIndex}
			rf.commitIndex++
			rf.applyCh <-applyMsg
			rf.debug("Sent a response to the end client 1 ")
			rf.debug("applyMsg of the LEADER %v",applyMsg)
			break
		}
		//Everybody has responded. Commitment could not be reached.
		if count == len(rf.peers)-1 {
			rf.debug("Everybody has responded. Commitment could not be reached.")
			break
		}
	}
	//Check at the last. This is because this way the first HB will be sent immediately.
	timer := time.NewTimer(100 * time.Millisecond)
	<-timer.C
}
//
// the tester calls Kill() when a Raft instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (rf *Raft) Kill() {
	// Your code here, if desired.
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
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	/// Start as a follower.
	rf.currentState = "FOLLOWER"
	rf.commitIndex=1
	rf.lastApplied=1 // Initializing this to 1 as we have added dummy 0th entry
	rf.votedFor=-1
	//05/12
	//Let the leader start with 0. When a candidate transitions to the candidate state it increments this value.
	rf.currentTerm=0
	//Initialize the log.
	//This is a dummy entry
	rf.log = append(rf.log, LogEntry{LastLogTerm: 0})
	rf.applyCh = applyCh
	rf.debug("++++++++++++++++++++++++++Length of the log during initialization---> %d \n",len(rf.log))
	rf.electionTimer = time.NewTimer((400 + time.Duration(rand.Intn(300))) * time.Millisecond)
	// Your initialization code here (2A, 2B, 2C).
	go rf.conductElection()
	//Send heart beat to everybody else
	return rf
}

func (rf* Raft)conductElection(){

	rf.debug("Inside conductElection ")
	count := len(rf.peers)-1//Count of peers. I should receive these many votes.
	//rf.debug("Count ")
	<-rf.electionTimer.C
	//Let us reset the timer here itself. This way when we don't get a majority we will save some time
	rf.resetElectionTimer()
	// When my timer goes off, I need to see whether I need to conduct election.
	rf.transitionToCandidate()
	lastIndex, lastTerm := rf.getLastEntryInfo()
	requestVoteArgs := RequestVoteArgs{Term:rf.currentTerm,CandidateId:rf.me,LastLogTerm:lastTerm,LastLogIndex:lastIndex }
	var voteCount = 1
	if rf.currentState!=Leader {
		votesCh := make(chan bool)
		for id := range rf.peers {
			if id != rf.me {
				//fmt.Println("Inside Go routine",id)
				go func(id int, peer *labrpc.ClientEnd) {
					requestVoteReply := RequestVoteReply{}
					rf.debug(" Before sending the request vote to %d ",id)
					ok := rf.sendRequestVote(id, &requestVoteArgs, &requestVoteReply)
					response := ok && requestVoteReply.VoteGranted
					//Check now whether everything is OK. This is moved from outside as we are creating requestVoteReply within
					// Go routine.
					if requestVoteReply.Term > rf.currentTerm {
						fmt.Println("Got a higher current term from peer " ,rf.me," So breaking")
						rf.transitionToFollower(requestVoteReply.Term)
					}
					votesCh <- response
				}(id,rf.peers[id])
			}else{
				//fmt.Println("I am ",rf.me)
			}
		}
		//fmt.Println("len(rf.peers)   ",len(rf.peers))
		for {
			if count == 0 {
				rf.debug("Count == 0")
				rf.conductElection()
				break
			}
			hasPeerVotedForMe := <-votesCh
			//I got  a response. I am going to decrement count
			count--
			if rf.currentState == Follower {
				break
			}
			rf.debug("Did I receive vote ? --> %t, currentVoteCount =%d ",hasPeerVotedForMe,voteCount)
			if hasPeerVotedForMe {
				voteCount +=1
				//fmt.Println(rf.me ," Incremented vote count-->",voteCount)
				//rf.debug()
				if voteCount > (len(rf.peers)/2) {
					//fmt.Println("I won the election !!! ",rf.me,"Vote count -->",voteCount, " ",len(rf.peers)/2)
					rf.debug("I won the election !!! VoteCount=%d, threshold = %d",voteCount,len(rf.peers)/2)
					go rf.promoteToLeader()
					break
				}
			}
		}

	}
}

func (rf* Raft) promoteToLeader(){
	rf.currentState =Leader
	rf.sendHeartBeat()
}

func (rf* Raft)sendHeartBeat(){
	for {
		if rf.currentState!=Leader{
			break
		}
		//Otherwise we will have to send heart beat for every peer.
		for id, peer := range rf.peers {
			//fmt.Println("peer -->",peer)
			if id != rf.me {
				go func(id int, peer *labrpc.ClientEnd) {
					var prevLogIndex, prevLogTerm = 0, 0
					if len(rf.log) > 0 {
						lastEntry := rf.log[len(rf.log)-1]
						prevLogIndex, prevLogTerm = lastEntry.LastLogIndex, lastEntry.LastLogIndex
					} else {
						prevLogIndex, prevLogTerm = 0, 0
					}
					reply := AppendEntriesReply{}
					args := AppendEntriesArgs{
						Term:             rf.currentTerm,
						LeaderID:         rf.me,
						PreviousLogIndex: prevLogIndex,
						PreviousLogTerm:  prevLogTerm,
						LogEntries:       make([]LogEntry, 0), //Empty array
						LeaderCommit:     rf.commitIndex,
					}

					requestName := "Raft.AppendEntries"
					ok := rf.peers[id].Call(requestName, &args, &reply)
					//fmt.Println("Called APPEND ENTRIES ***************** ",ok, " ",id)
					//rf.debug("Called APPEND ENTRIES ***************** OK = %d",ok)
					///If everything is ok or not
					// if term>myTerm => Transition to follower.
					if ok && reply.Term>rf.currentTerm {
						rf.mu.Lock()
						rf.transitionToFollower(reply.Term)
						rf.mu.Unlock()
					}
				}(id,peer)
			}

		}
		//Check at the last. This is because this way the first HB will be sent immediately.
		timer := time.NewTimer(100 * time.Millisecond)
		<-timer.C

	}
}

type AppendEntriesArgs struct {
	// Your data here.
	Term int
	LeaderID int
	PreviousLogTerm int
	PreviousLogIndex int
	LogEntries []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	// Your data here.
	Term int
	Success bool
	NextIndex int
}
func (rf *Raft) transitionToCandidate() {
	rf.currentState = Candidate
	rf.currentTerm++
	rf.votedFor = rf.me
	rf.debug("Transition to candidate, term=%d", rf.currentTerm)
}

func (rf *Raft) transitionToFollower(newTerm int) {
	rf.currentState = Follower
	rf.currentTerm = newTerm
	rf.votedFor = -1
	rf.resetElectionTimer()

}
func (rf* Raft) resetElectionTimer(){
	rf.debug("Restarting my timer")
	//fmt.Println("Restarting my timer ",rf.me)
	rf.electionTimer.Stop()
	rf.electionTimer.Reset((400 + time.Duration(rand.Intn(300))) * time.Millisecond)
}

func (rf *Raft) getLastEntryInfo() (int, int) {
	if len(rf.log) > 0 {
		entry := rf.log[len(rf.log)-1]
		return entry.LastLogIndex, entry.LastLogTerm
	}
	return 0,0
}

func (rf *Raft) debug(format string, a ...interface{}) {
	// NOTE: must hold lock when this function is called!
	Dprintf(rf.me, rf.currentState, format, a...)
	return
}

func (rf *Raft) printSlice(s []LogEntry, str string) {
	rf.debug ("%s -->",str)
	rf.debug(" length=%d capacity=%d %v\n", len(s), cap(s), s)
}