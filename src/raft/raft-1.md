<center><h1>Raft Implementation</h1></center>
<center><strong>Shuai Liu, MG1733039</strong></center>
<hr/>

# Abstract
In this big data time, high performance distributed systems are required to process the large volumn of data. However, it is not easy to organize plenty of nodes. One of the significant problems is distributed consensus, which means every node in the cluster will eventually reach a consensus without any conflict.

Raft is a distributed consensus algorithm which has been proved workable. This expriment mainly focus on designing and implementing leader election described in rart algorithm.

## keywords
Distributed Consensus, Leader Election, Log Replication


# Introduction

Before Raft, (multi-)Paxos has been treated as an industry standard for a long time. However, even with the help of Paxos, we still find it hard to build up a reliable distributed system.

Just as the comment from Chubby implementers:

> There are significant gaps between the description of the Paxos algorithm and the needs of a real-world system.... the final system will be based on an unproven protocol.

Paxos is rather difficult to implement mainly because it is not easy to understand for those who are not mathematicians.

Then Raft came out, which has a good understandability. Compared with Paxos, there are smaller state space achieved by reducing states. Also, Raft decomposes the problem into leader election, log replication, safety and membership changes, instead of treating them as a total of mess.

To further understand distributed consensus, this expriment tries to implement the first section _leader election_ in Raft and leave the rest parts in the following expriments.

# Design & Realization
The language we use in this expriment is Go 1.9.

## Structure
There are some states such as current term, logs, role of node that have to be stored and shared across the threads, so we design a structure called _Raft_. Among the variables, _currentTerm_, _votedFor_ and _logs_ are required to be persisted while the rest are volatile.

```go
type Raft struct {
	mu        sync.Mutex
	peers     []*labrpc.ClientEnd
	persister *Persister
	me        int // index into peers[]

	currentTerm int
	votedFor int
	logs []map[string]int

	commitIndex int
	lastApplied int

	nextIndex []int
	matchIndex []int

	role int //0-follower, 1-candidate, 2-leader
	electionTimeout time.Duration
}
```

## Initialization & main loop

When a node starts up, it firstly initializes the Raft object, generates election timeout and heart beat timeout randomlly. Then, it reads persisted data which are stored before last crash. After initailazation, it comes to the infinite main loop. In the main loop, the node sleeps for a certain time, when it wakes up, it checks whether he is the leader or the leader has connected with him not long ago (within _heartBeatTimeout_). If not, it increases the _currentTerm_ and starts a new election in another new thread.

There are two things to mention. The first is random _electionTimeout_. Same _electionTimeout_ may cause infinite elections if each node starts up at nearly the same time. Different timeout can help reduce the conflicts. To further reduce the conflicts, the node will re-generate _electionTimeout_ randomlly before an election for some nodes may have same _electionTimeout_ and may cause infinite elections especially in small clusters. The other is fake sleep. There is no easy approaches to extend the sleep time when the thread is in sleep. So the node will wake up earlier than expected. To emulate the delay, when the node wakes up, we will compare the time last heart beat arrived with current time, if the duration is smaller than _electionTimeout_, we make the node resume sleeping for the duration.

```go
	/* initialization */
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.electionTimeout = time.Duration(rand.Intn(100) + 100) * time.Millisecond
	rf.heartBeatTimeout = time.Duration(rand.Intn(50) + 50) * time.Millisecond
	rf.counter = 0
	rf.lastHeart = time.Now()
	rf.heartBeatTicker = time.NewTicker(rf.heartBeatTimeout)
	rf.role = 0 //start in follower state
	rf.commitIndex = 0
	rf.lastApplied = 0
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	go func() { //main loop
		for ;;{
			/* wait timeout */
			time.Sleep(rf.electionTimeout - time.Since(rf.lastHeart))

			if rf.role != 2 && time.Since(rf.lastHeart) >= rf.electionTimeout {
				rf.mu.Lock()
				rf.electionTimeout = time.Duration(rand.Intn(100) + 100) * time.Millisecond
				rf.currentTerm ++
				rf.votedFor = rf.me
				rf.role = 1
				rf.persist()
				rf.mu.Unlock()
				rf.doVote()
			}
			/* sleep at most electionTimeout duration */
			if time.Since(rf.lastHeart) >= rf.electionTimeout {
				rf.lastHeart = time.Now()
			}
		}
	}()
	return rf
}
```

<img alt="main loop" src="figures/DisEX02-1.png" style="zoom:50%" />

## Election
In the election, a node will exchange messages with other nodes. Two typical are request and response message of vote request. The term will keep until the leader crashed and another election are be made.

```go
type RequestVoteArgs struct {
	Term int
	CandidateId int
	LastLogIndex int
	LastLogTerm int
}
```

```go
type RequestVoteReply struct {
	Term int
	VoteGranted bool
}
```

The election performs as follows. The node who wants to be the leader firstly switch to candidate state and send a _RequestVote_ call to other nodes, attaching the term and logd infomation. The called node will compare the _Term_ with its _currentTerm_, it _Term_ is smaller, it simply refuse the request. Otherwise, the node will check whether if the logs of candidate is at least up to date with its _logs_. If all pass, it grants the request and switch to follower state and set _votedFor_ to _candidateId_. Whatever the result is, if higher _Term_ detected, the node will update _currentTerm_ to _Term_ and switch to follower state. In this expriment, there will be no logs appended, so the log checks is skipped.

When a majority of nodes agree the request, it switches to leader state and starts to send heart beat message to each node periodically until it is no longer a leader. If the heart beat reply or vote request reply reports a higher term, it will switch to follower state whatever the state is.

```go
func (rf *Raft) doVote() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	var agreed int64 = 1
	index := rf.commitIndex
	term := 0
	if index != 0 {
		term = rf.logs[index - 1]["term"]
	}
	for i:=0;i< len(rf.peers);i++{
		if i != rf.me {
			go func(peer int, currTerm int, index int, term int) {
				args := RequestVoteArgs{Term: currTerm, CandidateId:rf.me, LastLogIndex:index, LastLogTerm:term}
				reply := RequestVoteReply{}
				ok := rf.sendRequestVote(peer, args, &reply)
				rf.mu.Lock()
				if ok && args.Term == rf.currentTerm && rf.role == 1{
					atomic.AddInt64(&agreed, 1)
					if (int(agreed) * 2 > len(rf.peers)) {
						rf.role = 2
						/* persist state */
						rf.persist()
						for i:=0;i<len(peers);i++ {
							rf.nextIndex = append(rf.nextIndex, len(rf.logs) + 1)
							rf.matchIndex = append(rf.matchIndex, 0)
						}
						go rf.doSubmit()
					}
				}
				rf.mu.Unlock()
			}(i, rf.currentTerm, index, term)
		}
	}
}
```

```go
func (rf *Raft) sendRequestVote(server int, args RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	if reply.Term > rf.currentTerm {
		rf.mu.Lock()
		rf.currentTerm = reply.Term
		rf.role = 0
		rf.votedFor = -1
		rf.persist()
		rf.mu.Unlock()
	}
	return ok && reply.VoteGranted
}
```

```go
func (rf *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.VoteGranted = false
	} else {
		/* whatever, if higher term found, switch to follower */
		if args.Term > rf.currentTerm {
			rf.role = 0
			rf.votedFor = -1
			rf.currentTerm = args.Term
		}

		/* check log first, there is no requests in this expriment, so skip the details */
		if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) {
			reply.VoteGranted = true
			rf.role = 0
			rf.votedFor = args.CandidateId
			rf.lastHeart = time.Now()
		} else {
			reply.VoteGranted = false
		}
	}
	rf.persist()
	reply.Term = rf.currentTerm
}
```

## Heart beat
The heart beat message can be empty, but for conpatible with  following expriments, we set several variables in the request and response messages. As mentioned above, the the reply message reports a higher _Term_, the leader will immediatelly stop the heart beat and switch to follower state. When a node receives a heart beat message, it will update _lastHeart_ to _time.Now()_ to delay wake up.

```go
type AppendEntriesArgs struct {
	Term int
	LeaderId int
	PrevLogIndex int
	PrevLogTerm int
	Entries []map[string]int
	LeaderCommit int
}
```

```go
type AppendEntriesReply struct {
	Term int
	Success bool
}
```

```go
func (rf *Raft) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Success = false
	} else if args.Term > rf.currentTerm {
		reply.Success = true
		rf.currentTerm = args.Term
		rf.role = 0
		rf.votedFor = args.LeaderId
	} else {
		reply.Success = true
	}
	if reply.Success {
		rf.lastHeart = time.Now()
	}
	reply.Term = rf.currentTerm
}
```

```go
func (rf *Raft) sendAppendEntries(server int, args AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	if reply.Term > rf.currentTerm {
		rf.mu.Lock()
		rf.currentTerm = reply.Term
		rf.role = 0
		rf.votedFor = -1
		rf.persist()
		rf.mu.Unlock()
	}
	return ok && reply.Success
}
```

```go
func (rf *Raft) doSubmit() {
	/* ensure only one thread is running */
	if atomic.AddInt64(&rf.counter, 1) > 1 {
		atomic.AddInt64(&rf.counter, -1)
		return
	}
	for range rf.heartBeatTicker.C {
		if rf.role != 2 {
			break
		}
		for i:=0;i< len(rf.peers);i++{
			if i != rf.me {
				go func(peer int) {
					rf.mu.Lock()
					index := rf.nextIndex[peer]
					term := 0
					entries := make([]map[string]int, 0)
					args := AppendEntriesArgs{Term: rf.currentTerm, LeaderId:rf.me, PrevLogIndex:index - 1, PrevLogTerm:term, Entries:entries, LeaderCommit:rf.commitIndex}
					reply := AppendEntriesReply{}
					rf.mu.Unlock()

					ok := rf.sendAppendEntries(peer, args, &reply)
				}(i)
			}
		}
	}
	atomic.AddInt64(&rf.counter, -1)
}
```

During the implemention, we found some inadequates in Raft, one of them is the strategy of processing vote request. It says if a node meets a higher term, it will switch to follower state. Assume a cluster with three nodes A, B and C. A is the leader in term 1, and shortly B encounters a network failure whose duration is longer than _electionTimeout_, thus B will start a new term and request for votes. When the network resumes, A and C will detect a higher term from B's request and according the Raft algorithm they will stop this term and make a new election. However, the cluster works well and a new election is unnecessary. This means in a large cluster, even if only a single node restarts, the whole cluster has to be re-built. This will reduce the performance of the cluster to a large extent especially when the cluster is large.

Raft uses this strategy to make sure that leader can switch to follower state. But we can add another logic to realize that by counting followers. If a leader finds not a majority of nodes replies the heart beat message and it lasts longer than _electionTimeout_, it then switches to follower state and starts a new election.

# Validation
There two test cases _TestInitialElection_ and _TestReElection_ are designed to test the correctness of the system. Run the tests many times and the result shows our system passes all the test cases successfully.

<img alt="tests" src="figures/DisEX02-2.png" style="zoom:50%" />

# Conclusion
This expriment mainly focus on implementing the leader election part of Raft algorithm. The result shows that the cluster quickly generates a leader and remains the normal state until a failure, and after the failure the cluster can re-generate a new leader in a short time. This expriment proves the reliablity of Raft algorithm in another way.

_* The full and up-to-date code is hosted at https://github.com/newnius/NJU-DisSys-2017_

# References

[1] Ongaro D, Ousterhout J K. In search of an understandable consensus algorithm[C]//USENIX Annual Technical Conference. 2014: 305-319.

[2] [Raft](http://thesecretlivesofdata.com/raft/)

[3] [Go by Example: Atomic Counters](https://gobyexample.com/atomic-counters)

[4] [Go by Example: Non-Blocking Channel Operations](https://gobyexample.com/non-blocking-channel-operations)

[5] [Go by Example: Timers and Tickers](https://mmcgrana.github.io/2012/09/go-by-example-timers-and-tickers.html)

[6] [time: Timer.Reset is not possible to use correctly](https://github.com/golang/go/issues/14038)

[7] [论golang Timer Reset方法使用的正确姿势](http://tonybai.com/2016/12/21/how-to-use-timer-reset-in-golang-correctly/)

[8] [Go Channel 详解](http://colobu.com/2016/04/14/Golang-Channels/)

[9] [Go 语言切片(Slice)](http://www.runoob.com/go/go-slice.html)

[10] [Go 语言函数方法](https://wizardforcel.gitbooks.io/w3school-go/content/20-5.html)

[11] [Golang初学者易犯的三种错误](http://www.jianshu.com/p/e737cb26c141)

[12] [Not able to install go lang plugin on intellij 2017.2 community edition](https://github.com/go-lang-plugin-org/go-lang-idea-plugin/issues/2897)
