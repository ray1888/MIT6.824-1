package shardkv

import "shardmaster"

//
// Sharded key/value server.
// Lots of replica groups, each running op-at-a-time paxos.
// Shardmaster decides which group serves each shard.
// Shardmaster may change shard assignment from time to time.
//
// You will have to modify these definitions.
//

const (
	OK            = "OK"
	ErrNoKey      = "ErrNoKey"
	ErrWrongLeader = "ErrWrongLeader"
	ErrWrongGroup = "ErrWrongGroup"
	ErrTimeout    = "ErrTimeout"
	ErrNeedWait   = "ErrNeedWait"
)

type Err string

// Put or Append
type PutAppendArgs struct {
	// You'll have to add definitions here.
	Key   string
	Value string
	Op    string // "Put" or "Append"
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	ClientId 	  int64
	Seq   int
}

type PutAppendReply struct {
	WrongLeader bool
	Err         Err
}

type GetArgs struct {
	Key string
	// You'll have to add definitions here.
	ClientId  int64
	Seq int
}

type GetReply struct {
	WrongLeader bool
	Err         Err
	Value       string
}

type PullArgs struct {
	Shard       []int
	ClientId    int64
	Seq			int
}

type PullReply struct {
	MapKV       map[string]string
	ShardDup    [shardmaster.NShards]map[int64]int
	WrongLeader bool
	Err			Err
}

type DelArgs struct {
	Shard    []int
	ClientId int64
	Seq      int
}

type DelReply struct{
	WrongLeader bool
	Err         Err
}