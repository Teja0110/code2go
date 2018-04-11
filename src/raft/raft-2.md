<center><h1>Raft Implementation</h1></center>
<center><strong>Shuai Liu, MG1733039</strong></center>
<hr/>

# Abstract
In this big data time, high performance distributed systems are required to process the large volumn of data. However, it is not easy to organize plenty of nodes. One of the significant problems is distributed consensus, which means every node in the cluster will eventually reach a consensus without any conflicts.

Raft is a distributed consensus algorithm which has been proved workable. This expriment contitues the previous expriment and implements the log replication and finally tests the whole system in many abnormal situations.

## keywords
Distributed Consensus, Leader Election, Log Replication


# Introduction

Before Raft, (multi-)Paxos has been treated as an industry standard for a long time. However, even with the help of Paxos, we still find it hard to build up a reliable distributed system.

Just as the comment from Chubby implementers:

> There are significant gaps between the description of the Paxos algorithm and the needs of a real-world system.... the final system will be based on an unproven protocol.

Paxos is rather difficult to implement mainly because it is not easy to understand for those who are not mathematicians.

Then Raft came out, which has a good understandability. Compared with Paxos, there are smaller state space achieved by reducing states. Also, Raft decomposes the problem into leader election, log replication, safety and membership changes, instead of treating them as a total of mess.

To further understand distributed consensus, this expriment tries to implement the Raft algorithm in go language.

# Design & Realization
The language we use in this expriment is Go 1.9.

## Structure
There are some states such as current term, logs, role of node that have to be stored and shared across the threads, so we design a structure called _Raft_. Among the variables, _currentTerm_, _votedFor_ and _logs_ are required to be persisted while the rest are volatile. To save space, some variables for loop control are not mentioned here.

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

The main loop is almost the same as we mentioned in leader election in the previous expriment, so here we only discuss the differences.

## Election

In this expriment, we add following lines to the vote function, meaning inititialze _nextIndex_ and _matchIndex_ after being elected as the leader.

```go
rf.nextIndex = rf.nextIndex[0:0]
rf.matchIndex = rf.matchIndex[0:0]
for i:=0; i<len(rf.peers); i++ {
	rf.nextIndex = append(rf.nextIndex, len(rf.logs) + 1)
	rf.matchIndex = append(rf.matchIndex, 0)
}
```

Also, the process of handling vote request differs.

The server will compare _term_ of last log entry with candidate's last term, if the candidate claims a higher term or the two terms equal but candidate has larger log, it grants the request. Otherwise, it rejects.

Now, the _RequestVote_ function looks like follow.

```go
func (rf *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.VoteGranted = false
		fmt.Println(rf.me, "tells", args.CandidateId, ": you are to late, highest term is", rf.currentTerm)
	} else {
		/* whatever, if higher term found, switch to follower */
		if args.Term > rf.currentTerm {
			rf.role = 0
			rf.votedFor = -1
			rf.currentTerm = args.Term
			fmt.Println(rf.me, "says: higher term detected, term=", args.Term)
		}

		/* check log first */
		lastLogTerm := 0
		if len(rf.logs) > 0 {
			lastLogTerm = rf.logs[len(rf.logs) - 1]["term"];
		}
		if ((rf.votedFor == -1 || rf.votedFor == args.CandidateId) && (lastLogTerm < args.LastLogTerm || (lastLogTerm == args.LastLogTerm && len(rf.logs) <= args.LastLogIndex) )) {
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


## AppendEneries

The main process of log replication works as follows.

  - First of all, when a new command reaches the leader, the leader appendes it to _log_ and reply an index where command will exists if successfully replicated. Followers won't accept requests from clients, they simply redirect clients to the leader.
  - The leader sends new log entries to each server, attching index of previous log entry and the term of that entry.The log entries are determined by _nextIndex_.
  - Followers check whether they have previous log entries and then append new log entries to certain location.
  - If a majority of followers accept a command, the leader increases his _commitIndex_ and replies to the client.
  - The _commitIndex_ will be sent in next _appendEntry_ request, followers commit the commands known to have been replicated in a majority servers.

The format of request and reply.

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
	Len int
}
```

The leader sends _appendEntries_ request to each follower periodically to keep the role state. In a request, if a follower has log entries not replicated, the next log entries will be attached in the request. Each reuqest is executed in a new thread in case the network is slow or unreachable, which will block the loop.

Every _appendEntries_ contains more than one log entries. When a follower receives this request, it first confirms the role of the server claimed to be the server. Then it checks if it has already recorded the log entries before newer ones. If all pass, the server overwrite the log and replace log from certain index with given log entries in request.

The leader receives reply from followers and increase _matchIndex_, which means logs entries known to replicated in a follower. If a majority followers have replicated a log entry, the leader increase the _commitIndex_ by one and replies to the client. To prevent potential problems of unreliable network, the log entries are commited one by one in incremental order.

<img alt="tests" src="figures/DisEX03-2.png" style="zoom:80%" />

There is a potential problem in the log replication. If a leader receives a command but fails to replicate it due to network failure, it then becomes the leader in later term, now it can replicate the command to other followers. Unfortunatelly, it fails again shortyly after commiting the entry. In the origin algorithm, if a server is elected as the leader but that server does not contain that command, it will override the entry, and results in some followers commiting same log entry twice but with different commands. It conflicts with Raft rules that commands commited will exist in following leaders.

To prevent this, we extend the algorithm to not commit commands from older term immediatelly after a majority replicates. Instead, they are commited just after a command in current term to be commited. This ensures only server which contains the newest lon entry can be elected as the leader.

<img alt="tests" src="figures/DisEX03-3.png" style="zoom:80%" />

<img alt="tests" src="figures/DisEX03-4.png" style="zoom:80%" />

```go
func (rf *Raft) doSubmit() {
	/* ensure only one thread is running */
	if atomic.AddInt64(&rf.counter, 1) > 1 {
		atomic.AddInt64(&rf.counter, -1)
		return
	}
	for range rf.heartBeatTicker.C {
		if !rf.running {
			break
		}
		if rf.role != 2 {
			break
		}
		for i:=0;i< len(rf.peers);i++{
			if i != rf.me {
				go func(peer int) {
					rf.mu.Lock()
					index := rf.nextIndex[peer]
					term := 0
					/* TODO: limit entries max size */
					entries := make([]map[string]int, 0)
					if len(rf.logs) >= index {
						entries = append(entries, rf.logs[index-1:]...)
					}
					if index > 1 {
						//fmt.Println(index, rf.logs)
						term = rf.logs[index - 2]["term"]
					}
					args := AppendEntriesArgs{Term: rf.currentTerm, LeaderId:rf.me, PrevLogIndex:index - 1, PrevLogTerm:term, Entries:entries, LeaderCommit:rf.commitIndex}
					reply := AppendEntriesReply{}
					rf.nextIndex[peer] = args.PrevLogIndex + len(entries) + 1
					rf.mu.Unlock()

					ok := rf.sendAppendEntries(peer, args, &reply)

					rf.mu.Lock()
					if ok && args.Term == rf.currentTerm && rf.role == 2 {
						rf.matchIndex[peer] = args.PrevLogIndex + len(entries)
						/* update commitIndex */
						for iter:=rf.commitIndex; iter<len(rf.logs); iter++ {
							if rf.logs[iter]["term"] < rf.currentTerm {
								continue
							}
							count := 1
							for j:=0;j<len(rf.peers);j++ {
								if j != rf.me {
									if rf.matchIndex[j] > iter {
										count++
									}
									//fmt.Println(j, rf.matchIndex[j], count)
								}
							}

							if count * 2 > len(rf.peers){
								for i:=rf.commitIndex; i<=iter; i++ {
									rf.commitIndex = i + 1
									command := rf.logs[i]["command"]
									fmt.Println(rf.me, "says: ", command, "is committed, index=", i + 1)
									rf.applyCh <- ApplyMsg{Index: i + 1, Command: command, UseSnapshot:false, Snapshot:rf.persister.ReadRaftState()}
								}
							} else { // commit in order
								break
							}
						}
					}
					rf.mu.Unlock()
				}(i)
			}
		}
	}
	fmt.Println(rf.me, "says: stop heart beat")
	atomic.AddInt64(&rf.counter, -1)
}
```

```go
func (rf *Raft) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		reply.Success = false
	} else {
		reply.Success = true
		rf.currentTerm = args.Term
		rf.role = 0
		rf.votedFor = args.LeaderId

		if args.PrevLogIndex > 0 {
			if len(rf.logs) >= args.PrevLogIndex && rf.logs[args.PrevLogIndex-1]["term"] == args.PrevLogTerm {
				reply.Success = true
			} else {
				reply.Success = false
				reply.Len = len(rf.logs)
				rf.lastHeart = time.Now()
			}
		}
	}
	reply.Term = rf.currentTerm
	if reply.Success {
		rf.logs = rf.logs[0:args.PrevLogIndex]
		//sync logs && apply
		rf.logs = append(rf.logs, args.Entries...)
		for iter:=rf.commitIndex; iter < args.LeaderCommit; iter++ {
			command := rf.logs[iter]["command"]
			rf.applyCh <- ApplyMsg{Index: iter + 1, Command:command, UseSnapshot:false, Snapshot:rf.persister.ReadRaftState()}
		}
		rf.commitIndex = args.LeaderCommit
		rf.lastHeart = time.Now()
	}
	rf.persist()
}
```

```go
func (rf *Raft) sendAppendEntries(server int, args AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	if ok {
		if reply.Term > rf.currentTerm {
			rf.mu.Lock()
			rf.currentTerm = reply.Term
			rf.role = 0
			rf.votedFor = -1
			rf.persist()
			rf.mu.Unlock()
		}
		if ok && rf.role == 2 && !reply.Success {
			rf.nextIndex[server] = args.PrevLogIndex
			if reply.Len < args.PrevLogIndex {
				rf.nextIndex[server] = reply.Len + 1
			}
		}
	}
	return ok && reply.Success
}
```

In some cases, a follower may lose log entries and as a result it can not simply override the log. In these cases, it replies false in the _appendEntries_ request, the leader will then decrease the _nextIndex_ and sends older log entries until the missing logs entries are fixed.

However, a server may lose too many log entries. If we use the above strategy, it consumes a long time. To speed up the decreament, we add a parameter _len_ to the reply structure which means the log longth of follower. Then the leader can reset _nextIndex_ to _len_ to reduce the time re-trying. Or we can use a snapshot.

# Validation

There are a total 17 test case _TestInitialElection_, _TestReElection_, _TestBasicAgree_, _TestFailAgree_, _TestFailNoAgree_, _TestConcurrentStarts_, _TestRejoin_, _TestBackup_, _TestCount_, _TestPersist1_, _TestPersist2_, _TestPersist3_, _TestFigure8_, _TestUnreliableAgree_, _TestFigure8Unreliable_, _TestReliableChurn_ and _internalChurn_, ranging from normal state to unreliable network such as network delay, network partition, package loss, duplicated packages and reordering of packages.

Run the tests many times and the result shows that our system passes all the test cases successfully.

<img alt="tests" src="figures/DisEX03-1.png" style="zoom:50%" />

This is the output of _TestBasicAgree_

```
0 says: hello world!
1 says: hello world!
2 says: hello world!
3 says: hello world!
4 says: hello world!
Test: basic agreement ...
1 says: I am not a leader
2 says: I am not a leader
3 says: I am not a leader
4 says: I am not a leader
0 says: I am not a leader
2 says: bye~
0 says: stop heart beat
1 says: I am not a leader
2 says: I am not a leader
3 says: I am not a leader
4 says: I am not a leader
0 says: I am not a leader
0 says: bye~
1 says: I am not a leader
2 says: I am not a leader
3 says: I am not a leader
4 says: I am not a leader
0 says: I am not a leader
1 says: bye~
1 tells 2 : vote me,  {28 1 0 0}
1 tells 0 : vote me,  {28 1 0 0}
0 says: higher term detected, term= 28
0 tells 1 : vote granted
2 says: higher term detected, term= 28
2 tells 1 : vote granted
1 says: I am the leader in term 28
1 says: stop heart beat
3 tells 0 : vote me,  {1 3 0 0}
3 tells 4 : vote me,  {1 3 0 0}
3 tells 1 : vote me,  {1 3 0 0}
3 tells 2 : vote me,  {1 3 0 0}
0 says: higher term detected, term= 1
0 tells 3 : vote granted
4 says: higher term detected, term= 1
4 tells 3 : vote granted
1 says: higher term detected, term= 1
1 tells 3 : vote granted
2 says: higher term detected, term= 1
2 tells 3 : vote granted
3 says: I am the leader in term 1
3 tells 4 : ping, {1 3 0 0 [] 0}
3 tells 1 : ping, {1 3 0 0 [] 0}
3 tells 2 : ping, {1 3 0 0 [] 0}
3 tells 0 : ping, {1 3 0 0 [] 0}
4 tells 3 : pong, &{1 true}
2 tells 3 : pong, &{1 true}
1 tells 3 : pong, &{1 true}
0 tells 3 : pong, &{1 true}
1 says: I am not a leader
2 says: I am not a leader
3 says: new command 100 in term 1
3 tells 2 : ping, {1 3 0 0 [map[command:100 term:1]] 0}
3 tells 0 : ping, {1 3 0 0 [map[command:100 term:1]] 0}
3 tells 1 : ping, {1 3 0 0 [map[command:100 term:1]] 0}
3 tells 4 : ping, {1 3 0 0 [map[command:100 term:1]] 0}
1 tells 3 : pong, &{1 true}
2 tells 3 : pong, &{1 true}
4 tells 3 : pong, &{1 true}
0 tells 3 : pong, &{1 true}
3 says:  100 is committed, index= 1
3 tells 4 : ping, {1 3 1 1 [] 1}
3 tells 2 : ping, {1 3 1 1 [] 1}
3 tells 1 : ping, {1 3 1 1 [] 1}
3 tells 0 : ping, {1 3 1 1 [] 1}
2 says: commit 100 index= 1
4 says: commit 100 index= 1
2 tells 3 : pong, &{1 true}
4 tells 3 : pong, &{1 true}
1 says: commit 100 index= 1
1 tells 3 : pong, &{1 true}
0 says: commit 100 index= 1
0 tells 3 : pong, &{1 true}
1 says: I am not a leader
2 says: I am not a leader
3 says: new command 200 in term 1
3 tells 1 : ping, {1 3 1 1 [map[command:200 term:1]] 1}
3 tells 2 : ping, {1 3 1 1 [map[command:200 term:1]] 1}
3 tells 4 : ping, {1 3 1 1 [map[command:200 term:1]] 1}
3 tells 0 : ping, {1 3 1 1 [map[command:200 term:1]] 1}
1 tells 3 : pong, &{1 true}
4 tells 3 : pong, &{1 true}
2 tells 3 : pong, &{1 true}
0 tells 3 : pong, &{1 true}
3 says:  200 is committed, index= 2
3 tells 4 : ping, {1 3 2 1 [] 2}
3 tells 0 : ping, {1 3 2 1 [] 2}
3 tells 1 : ping, {1 3 2 1 [] 2}
3 tells 2 : ping, {1 3 2 1 [] 2}
0 says: commit 200 index= 2
0 tells 3 : pong, &{1 true}
2 says: commit 200 index= 2
4 says: commit 200 index= 2
2 tells 3 : pong, &{1 true}
1 says: commit 200 index= 2
1 tells 3 : pong, &{1 true}
4 tells 3 : pong, &{1 true}
1 says: I am not a leader
2 says: I am not a leader
3 says: new command 300 in term 1
3 tells 1 : ping, {1 3 2 1 [map[command:300 term:1]] 2}
3 tells 2 : ping, {1 3 2 1 [map[term:1 command:300]] 2}
3 tells 0 : ping, {1 3 2 1 [map[command:300 term:1]] 2}
3 tells 4 : ping, {1 3 2 1 [map[command:300 term:1]] 2}
1 tells 3 : pong, &{1 true}
2 tells 3 : pong, &{1 true}
3 says:  300 is committed, index= 3
4 tells 3 : pong, &{1 true}
0 tells 3 : pong, &{1 true}
3 tells 4 : ping, {1 3 3 1 [] 3}
3 tells 1 : ping, {1 3 3 1 [] 3}
3 tells 2 : ping, {1 3 3 1 [] 3}
3 tells 0 : ping, {1 3 3 1 [] 3}
4 says: commit 300 index= 3
4 tells 3 : pong, &{1 true}
1 says: commit 300 index= 3
2 says: commit 300 index= 3
0 says: commit 300 index= 3
1 tells 3 : pong, &{1 true}
2 tells 3 : pong, &{1 true}
0 tells 3 : pong, &{1 true}
  ... Passed

```


# Conclusion
This expriment implements the rest parts of Raft and then makes a fully test on the whole system. The result shows that the cluster quickly generates a leader and remains the normal state until a failure, and after the failure the cluster can re-elect a new leader in a short time. Even in some extremely bad network situations, the system can tolerant the unreliable network and works well. This expriment proves the reliablity of Raft algorithm in another way.

What's more, this expriment shows that test-driven development has great value, it can expose potential problems which is not easy to find by code review.

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
