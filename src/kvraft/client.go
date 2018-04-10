package raftkv

import "labrpc"
import "crypto/rand"
import (
	"math/big"
	"time"
	"fmt"
	"strconv"
)


type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	id int
	cnt int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	// You'll have to add code here.
	fmt.Println("MakeClerk")
	ck.id = 0
	ck.cnt = 0
	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("RaftKV.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) Get(key string) string {
	value := ""
	success := false
	for !success {
		for i:=0;i<len(ck.servers);i++ {
			//fmt.Println("Call", i)
			args := GetArgs{Key:key, UUID: strconv.Itoa(ck.id) + "_" + strconv.Itoa(ck.cnt)}
			reply := &GetReply{}
			ok := ck.servers[i].Call("RaftKV.Get", &args, reply)
			if ok && !reply.WrongLeader {
				success = true
				value = reply.Value
				if ck.id == 0 {
					ck.id = reply.ID
				}
				break
			}
		}
		time.Sleep(time.Millisecond * 10)
	}
	ck.cnt += 1
	return value
}

//
// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("RaftKV.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) PutAppend(key string, value string, opt string) {
	if ck.id == 0 {
		ck.Get("nobody")
	}

	fmt.Println(opt, key, value, "--------------------------")
	success := false
	for !success{
		for i:=0;i<len(ck.servers);i++ {
			args := PutAppendArgs{Key: key, Opt: opt, Value:value, UUID: strconv.Itoa(ck.id) + "_" + strconv.Itoa(ck.cnt)}
			reply := &PutAppendReply{}
			ok := ck.servers[i].Call("RaftKV.PutAppend", &args, reply)
			if ok &&reply.WrongLeader == false {
				success = true
				fmt.Println(opt, key, value, "success")
			}
		}
		time.Sleep(time.Millisecond * 200)
	}
	ck.cnt += 1
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
