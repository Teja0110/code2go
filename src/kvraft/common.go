package raftkv

const (
	OK       = "OK"
	ErrNoKey = "ErrNoKey"
)

type Err string

// Put or Append
type PutAppendArgs struct {
	// You'll have to add definitions here.
	Key   string
	Value string
	Opt    string
	UUID string
}

type PutAppendReply struct {
	WrongLeader bool
	Err         Err
	ID int
}

type GetArgs struct {
	Key string
	// You'll have to add definitions here.
	UUID string
}

type GetReply struct {
	WrongLeader bool
	Err         Err
	Value       string
	ID int
}
