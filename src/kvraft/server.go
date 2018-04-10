package raftkv

import (
	"encoding/gob"
	"labrpc"
	"log"
	"raft"
	"sync"
	"fmt"
	"time"
)

const Debug = 1

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}


type Op struct {
	Opt string
	Key string
	Value string

	UUID string
}


type RaftKV struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	currentIndex int
}


func (kv *RaftKV) Get(args *GetArgs, reply *GetReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	op := Op{Opt:"Get", Key:args.Key, UUID:args.UUID}
	index, _, isLeader := kv.rf.Start(op)
	reply.WrongLeader = true
	if isLeader {
		fmt.Println(kv.me, "Get", args.Key, "")
		/* wait until success */
		cnt := 0
		for {
			//fmt.Println(kv.me, "waiting", index)
			cnt += 1
			if cnt > 30 {
				break
			}
			if kv.currentIndex == index {
				fmt.Println("Get success at index", index)
				_, logs, commitIndex := kv.rf.GetState2()
				var db map[string]string
				db = make(map[string]string)

				var UUIDs map[string]int
				UUIDs = make(map[string]int)


				fmt.Println("logs...")
				for i:=0;i<commitIndex;i++{
					op := logs[i]["command"].(Op)
					fmt.Println(i, "=>", op)
					/* check duplicates */
					if op.Opt != "Get" && UUIDs[op.UUID] > 0 {
						fmt.Println("skip", op)
						continue
					}
					UUIDs[op.UUID] += 1

					switch op.Opt {
					case "Get":
						break
					case "Put":
						db[op.Key] = op.Value
						break
					case "Append":
						db[op.Key] = db[op.Key] + op.Value
						break
					}
				}
				fmt.Println("logs end...")
				fmt.Println(kv.me, "Get", args.Key, "value:", db[args.Key])
				reply.WrongLeader = false
				reply.Value = db[args.Key]
				reply.ID = index
				break
			}
			time.Sleep(time.Millisecond * 10)
		}
	}
}

func (kv *RaftKV) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	op := Op{
		Opt:args.Opt, Key:args.Key, Value:args.Value, UUID: args.UUID}
	index, _, isLeader := kv.rf.Start(op)
	cnt := 0
	reply.WrongLeader = true
	if isLeader {
		fmt.Println(kv.me, args.Opt, args.Key, args.Value)
		/* wait until success */
		for {
			//fmt.Println(kv.me, "waiting", index)
			cnt += 1
			if cnt > 500 {
				break
			}
			//fmt.Println("currentIndex:", kv.currentIndex)
			if kv.currentIndex >= index {
				_, logs, _ := kv.rf.GetState2()
				tmp := logs[index - 1]["command"].(Op)
				if tmp.Opt == op.Opt && tmp.Key == op.Key && tmp.Value == op.Value {
					fmt.Println(kv.me, args.Opt, args.Key, "success at index", index)
					reply.WrongLeader = false
					reply.ID = index
				}
				break
			}
			time.Sleep(time.Millisecond * 10)
		}
	}
}

//
// the tester calls Kill() when a RaftKV instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *RaftKV) Kill() {
	kv.rf.Kill()
	// Your code here, if desired.
	fmt.Println(kv.me, "Killed")
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots with persister.SaveSnapshot(),
// and Raft should save its state (including log) with persister.SaveRaftState().
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *RaftKV {
	// call gob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	gob.Register(Op{})

	kv := new(RaftKV)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// Your initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)
	fmt.Println(kv.me, "StartKVServer")

	go func() {
		//fmt.Println("start sub")
		for {
			msg := <-kv.applyCh
			if msg.Index > kv.currentIndex {
				kv.currentIndex = msg.Index
			}
			//fmt.Println(kv.me, msg)
		}
		//fmt.Println("finish")
	}()

	return kv
}
