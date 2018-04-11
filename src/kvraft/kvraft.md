<center><h1>KVRaft: Key-Value Storage Service based on Raft</h1></center>
<center><strong>Shuai Liu, MG1733039</strong></center>
<hr/>


# Abstract
RDBMS has long been an oracle in the field of database, but actually there are weakness for RDBMS to fit some situations where simple key value service is more appropriate.

In this experiment, we build up a key-value storage service named KVRaft based on distributed consensus framework Raft and fully validate the robust and perfomance of KVRaft including in extreme situations.

## keywords
Key-Value database, Distributed, NoSQL

# Introduction

RDBMS has been the only choice for a long time in the field of database, however, RDMBS can be too redundant in many cases like cache for example. Thus, a new kind of storage service called key-value storage occured. By removing unecessary parts and re-design the structure, a key-value storage service can achieve much higher performance than RDBMS.

In this experiment, we try to implement a key-value storage service named KVRaft based on our previous work Raft. The requirements we want to get from KVRaft are:

  - Each server coordinates with each other only by Raft logs
  - Even if a minority servers fail, the system can still work fine
  - The result has to be completely correct

The basic idea of KVRaft is, logging the changes into the Raft logs and rebuilding the state machine from the logs when a client makes a query.

# Model

The main architecture of KVRaft looks like this

<img alt="KVRaft Architecture" src="figures/kvraft-architecture.png" style="zoom:40%" />

Each KVRaft server contains a Raft server in its process, and the KVRaft servers don't coordinate with each other directly, they communicate with others only through the Raft logs. When a client wants to make a request, it tries the KVRaft servers one by one until one of the servers accepts the request. When receiving a request from a client, the KVRaft server firstly check whether the related Raft server is the leader in the Raft cluster currently, and accepts the request if true. That server then asks the Raft server to log the change and waits until the change is successfully distributed to the Raft servers.

There are three kinds of requests in the KVRaft storage service, _Get_, _Put_ and _Append_, and the requests are made by RPC. A _Put k v_ request sets the value of key _k_ to _v_, an _Append k v_ request sets the value of _k_ to be _Vold + v_, and a _Get k_ request queries the value of key _k_. If an _Append_ request tries to update a key which is not exist, it acts like a _Put_ request.

# Fault Tolerance

In a distributed system, failure is considered usual, so the KVRaft system should handle many kinds of failures.

## Retry
There are many cases where requests cannot be processed such as network disconnection, network partition, non-leader. In these situations, if the client doesn't receive a posotive response from the KVRaft server, it then goes on tring the next server until the request is accepted. A KVRaft server will wait for some time before it receives a success message from the Raft server. If timeouts, the KVRaft server will return error to the client so that the client knows the request fails and will try next server.

By retrying, a request can be accpted without infinite waiting even when a minority of serves fail, and the _timeout_ can be set to a bit longer than than _revote timeout_.

## Identity the real leader
Due to the weakness of Raft as I mentioned in the previous works, a Raft server cannot know it has already lost the leadership, thus it will still act like the leader and accept requests. In KVRaft, if a KVRaft server receives a query request, it doesn't know actually it is not the real leader currently, neither the Raft server. In this situation, the KVRaft server will accept the request and may generate a wrong value as a result.

Consider that _Put_ and _Append_ won't encounter this error because they will be loged into the Raft logs. If a Raft server is not the real leader, it cannot commit the log successfully. So the easist way to identity whether the Raft server is the real server, we can log the _Get_ request into the Raft logs as well, so that we can say the Raft server must be the leader if that server successfully commits the _Get_.

## Handle duplicate
When a failure encountered, the client will retry next server until the request is accepted. Let's think about a situation where a leader accepts the request but loses the leadership before commiting it, so the request will be sent to the next leader and that leader will log that request and commit it, if then the previous leader is re-voted as the leader it will commit the same request. This would cause duplicate log entries in the Raft logs and it cannot be avoided by the Raft servers.

One way to handle that is by modifing the request and append the last request the client makes, so that in the progress of rebuild, the KVRaft server can ignore duplicate requests by the orders. However, there is a problem. If the first request of a client fails and it raises duplicate log entries, it is unable to find them as there is no requests ahead. Also, two clients may make same requests and thus would cause potential errors because it is hard to determine the origin sequence.

To prevent the first situation, we can make a _Get_ request at first for each client. To fully solve the duplicate problem clearlly, we use the UUID to identity each request, when rebuilding the logs, just ignore the requests which has already been processed.

The UUID consists of ClientID and IncrementialID of each client. The IncrementialID starts with 1 and increments by one for each request. To generate a universal ClientID, we need another system which would arouse other problems. But fortunatelly, in this experiment, we can get the index of the log where the request would appear. The index is universal unique and we can use the index as the ClientID. Every time a new client is created, it first check if it is the first time to make a request and if so it would make a _Get_ request to receive the ClientID. The only addon for this change is another _Get_ request for each client and the cost is acceptable.

# Implementation

## Modify Raft
In our previous experiments, we assume the _command_ is an integer, so we have to update the related parts to make the Raft servers support non-integer commands by modifing the interface.

```golang
logs []map[string]interface{}
```

The other modification is to expose the logs to the KVRaft server by adding a function _getState2_ which would return the logs and whether the server is the leader. 

```golang
func (rf *Raft) GetState2() (bool, []map[string]interface{}, int) {
	return rf.role==2, rf.logs, rf.commitIndex
}
```

## Define RPC
There are three kinds of requests in KVRaft, _Get_, _Put_ and _Append_. we can combine _Put_ and _Append_ so there would be two RPCs.

#### _PutAppend_
```
type PutAppendArgs struct {
	Key   string
	Value string
	Opt    string
	UUID string
}
```

```golang
type PutAppendReply struct {
	WrongLeader bool
}
```
When the server successfully submits the request, the _WrongLeader_ is set to false meaning the request is loged, otherwise false.

#### _Get_
```golang
type GetArgs struct {
	Key string
	UUID string
}
```

```golang
type GetReply struct {
	WrongLeader bool
	Value       string
	ID int
}
```

The _ID_ actually means where the request is in the Raft logs and will be used as _ClientID_ by the UUID part.

## Client
The client firstly checks if a _Get_ request is made previously and if not make one. Then it retries until a server responds positively.

```
func (ck *Clerk) PutAppend(key string, value string, opt string) {
	if ck.id == 0 {
		ck.Get("nobody")
	}
	success := false
	for !success{
		for i:=0;i<len(ck.servers);i++ {
			args := PutAppendArgs{Key: key, Opt: opt, Value:value,
				UUID: strconv.Itoa(ck.id) + "_" + strconv.Itoa(ck.cnt)}
			reply := &PutAppendReply{}
			ok := ck.servers[i].Call("RaftKV.PutAppend", &args, reply)
			success =  ! (ok && reply.WrongLeader)
		}
		time.Sleep(time.Millisecond * 200)
	}
	ck.cnt += 1
}
```


## KVraft server

When rebuilding the state machine, the KVRaft server firstly gets logs from the Raft server, and then iterates the log entries. If an update whose UUID is not applied already, then apply the update to the state machine.

```golang
func (kv *RaftKV) Get(args *GetArgs, reply *GetReply) {
	op := Op{Opt:"Get", Key:args.Key, UUID:args.UUID}
	index, _, isLeader := kv.rf.Start(op)
	reply.WrongLeader = true
	if isLeader {
		cnt := 0
		for {
			cnt += 1
			if cnt > 30 {
				break
			}
			if kv.currentIndex == index {
				_, logs, commitIndex := kv.rf.GetState2()
				var db map[string]string
				db = make(map[string]string)
				var UUIDs map[string]int
				UUIDs = make(map[string]int)
				for i:=0;i<commitIndex;i++{
					op := logs[i]["command"].(Op)
					if op.Opt != "Get" && UUIDs[op.UUID] > 0 {
						continue
					}
					UUIDs[op.UUID] += 1
					switch op.Opt {
					case "Put":
						db[op.Key] = op.Value
						break
					case "Append":
						db[op.Key] = db[op.Key] + op.Value
						break
					}
				}
				reply.WrongLeader = false
				reply.Value = db[args.Key]
				reply.ID = index
				break
			}
			time.Sleep(time.Millisecond * 10)
		}
	}
}
```

# Optimization

## Remember the leader
In reality, the Raft cluster is stable in most of the time which means the leader changes rarely. In our design, the client tries from the first server every time, but actually this is a waste of time. The leader should be remembered and tried first as the next request, only when the remembered leader loses its leadership, the client has to try another server.

```
try remembered leader x
if success:
	return
else:
	for i in range(1, n+1):
		try server (leader + i) % n
		if success:
			return
```

The servers are logically organized as a ring by the clients, this design can make sure every server would be tried at most once per request when the Raft cluster is still in service.

## Minimize log size
As time passes, the size Raft logs grows and would make it more time consuming to rebuild the state machine. If we look into the Raft logs, we can see many redundent log entries since we just want to rebuild the state machine, not the history of changing. We can rewrite the log entries periodically to minimize the size and speed up the rebuild process.

For example, if we have a slice as follows:

```
put k m
append k n
put k o
append k p
```

we can rewrite them to:
```
put k op
```

One way to do this in hot can be starting another thread to copy the logs and rewiting them asynchronously and then replacing the log entries using snapshot in Raft. Another thing to take care is that the way we generate ClientID, which is the index of log entry. By changing list to map can solve this problem.

## Ignore other keys
Every time the KVRaft server receives a query, it has to rebuild the state machine and the state machine cannot be cached or reused. Actually, the query request only needs one key and the others are unnecessary, so we can focus only on the required key and ignore other keys, this method can minimize the size of UUID set and speed up the rebuilding.

# Validation

## Robust

There are a total of 12 test cases for KVRaft, covering many kinds of extreme situations _TestBasic_, _TestConcurrent_, _TestUnreliable_, _TestUnreliableOneKey_, _TestOnePartition_, _TestManyPartitionsOneClient_, _TestManyPartitionsManyClients_, _TestPersistOneClient_, _TestPersistConcurrent_, _TestPersistConcurrentUnreliable_, _TestPersistPartition_ and _TestPersistPartitionUnreliable_.

Run the tests many times and the result shows that our system passes all the test cases successfully.

<img alt="validation" src="figures/kvraft-test.png" style="zoom:50%" />

One of the debug logs shows that KVRaft can successfully handle duplicate requests.

```
logs...
0 => {Get nobody  0_0}
1 => {Get nobody  0_0}
2 => {Get nobody  0_0}
3 => {Get nobody  0_0}
4 => {Get nobody  0_0}
5 => {Put 0  1_1}
6 => {Put 1  2_1}
7 => {Put 2  3_1}
8 => {Put 4  5_1}
9 => {Put 3  4_1}
10 => {Put 0  1_1}
skip {Put 0  1_1}
11 => {Append 1 x 1 0 y 2_2}
12 => {Append 2 x 2 0 y 3_2}
13 => {Get 4  5_2}
logs end...
```

## Delay

We can see from the test cases that each request normally takes several milliseconds to be processed, and only in some extreme situations such partition it will take longer. Overall, KVRaft work fine as a distributed key-value storage service.

# Conclusion
This experiment builds a key-value storage service KVRaft based on the distributed consensus gramework Raft and the result shows that KVRaft is realiable and robust. The average delay of requests can be limited within several milliseconds and most of the requests in extreme states can be processed within seconds.

_* The full and up-to-date code is hosted at https://github.com/newnius/NJU-DisSys-2017_

# Reference

[1] Ongaro D, Ousterhout J K. In search of an understandable consensus algorithm[C]//USENIX Annual Technical Conference. 2014: 305-319.

[2] [Raft](http://thesecretlivesofdata.com/raft/)

[3] [Go maps in action](https://blog.golang.org/go-maps-in-action)

[4] [Go by Example: Non-Blocking Channel Operations](https://gobyexample.com/non-blocking-channel-operations)

[5] [Go Channel 详解](http://colobu.com/2016/04/14/Golang-Channels/)

[6] [Universally unique identifier(UUID)](https://en.wikipedia.org/wiki/Universally_unique_identifier)




