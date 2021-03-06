package shardmaster

import (
	"raft"
	"sort"
)
import "labrpc"
import "sync"
import "labgob"
import "time"
import "fmt"

type Result struct {
	Opname string
	ClientId     int64
	Seq int
	config Config
}

const (
	Alive = iota
	Killed
)

type ShardMaster struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	// Your data here.

	configs []Config // indexed by config num
	detectDup map[int64]Result
	chanresult map[int]chan Result

	state   int
}


type Op struct {
	// Your data here.
	Opname  string

	Servers map[int][]string
	GIDs []int
	Shard int
	GID int
	Num int

	ClientId      int64
	Seq     int
}

func (sm *ShardMaster) MakeEmptyConfig() Config{
	res := Config{}
	res.Num = 0
	for i := 0; i < NShards; i++ {
		res.Shards[i] = 0
	}
	res.Groups = make(map[int][]string)
	return res
}

func (sm *ShardMaster) CopyConfig(c1 *Config, c2 *Config) {
	if c2 == nil {
		return
	}
	c1.Num = c2.Num
	for i := 0; i < NShards; i++ {
		c1.Shards[i] = c2.Shards[i]
	}
	c1.Groups = make(map[int][]string)
	if c2.Groups == nil {
		return
	}
	for k, v := range c2.Groups {
		c1.Groups[k] = make([]string, len(v))
		for i := 0; i < len(v); i++ {
			c1.Groups[k][i] = v[i]
		}
	}
}

func (sm *ShardMaster) DupCheck(id int64, seq int) bool {
	res, ok := sm.detectDup[id]
	if ok{
		return seq > res.Seq
	}
	return true
}

func (sm *ShardMaster) CheckSame(c1 Op, c2 Op) bool {
	if c1.ClientId == c2.ClientId && c1.Seq == c2.Seq{
		return true
	}
	return false
}

func (sm *ShardMaster) StartCommand(op Op) (Err, Config){
	sm.mu.Lock()
	if !sm.DupCheck(op.ClientId, op.Seq){
		resCig := sm.MakeEmptyConfig()
		if op.Opname == "Query" {
			num := op.Num
			if op.Num <0 || op.Num >= len(sm.configs) {
				num = len(sm.configs) - 1
			}
			res := sm.configs[num]
			sm.CopyConfig(&resCig, &res)
			fmt.Println(sm.me, "query duplicate, num", num, ", shard config", resCig.Shards,", client ", op.ClientId,", seq", op.Seq)
		}
		sm.mu.Unlock()
		return OK, resCig
	}

	index, _, isLeader := sm.rf.Start(op)
	if !isLeader{
		sm.mu.Unlock()
		resCig := sm.MakeEmptyConfig()
		return ErrWrongLeader, resCig
	}
	//fmt.Println("index",index, sm.me, op)
	ch := make(chan Result, 1)
	sm.chanresult[index] = ch
	sm.mu.Unlock()
	defer func(){
		sm.mu.Lock()
		delete(sm.chanresult, index)
		sm.mu.Unlock()
	}()

	select{
	case c := <-ch:
		if c.ClientId == op.ClientId && c.Seq == op.Seq {
			resCig := sm.MakeEmptyConfig()
			sm.CopyConfig(&resCig, &c.config)
			return OK, resCig
		} else{
			resCig := sm.MakeEmptyConfig()
			return ErrWrongLeader, resCig
		}
		//fmt.Println("success ", op.Id, op.Seq)
	case <- time.After(time.Millisecond * 2000):
		fmt.Println("timeout")
		resCig := sm.MakeEmptyConfig()
		return ErrWrongLeader, resCig
	}
}

func (sm *ShardMaster) Join(args *JoinArgs, reply *JoinReply) {
	// Your code here.
	argser := make(map[int][]string)
	for k, v := range args.Servers {
		argser[k] = make([]string, len(v))
		for i := 0; i < len(v); i++ {
			argser[k][i] = v[i]
		}
	}
	op := Op{Opname:"Join",Servers:argser, ClientId:args.ClientId, Seq:args.Seq}
	err, _ := sm.StartCommand(op)

	reply.Err = err
	if err == ErrWrongLeader{
		reply.WrongLeader = true
	}else{
		reply.WrongLeader = false
	}
	if err == OK {
		fmt.Println(sm.configs[len(sm.configs) - 1])
	}
}

func (sm *ShardMaster) Leave(args *LeaveArgs, reply *LeaveReply) {
	// Your code here.
	arggid := make([]int, len(args.GIDs))
	for i := 0; i < len(args.GIDs); i++ {
		arggid[i] = args.GIDs[i]
	}
	op := Op{Opname:"Leave", GIDs:arggid, ClientId:args.ClientId, Seq:args.Seq}
	err, _ := sm.StartCommand(op)

	reply.Err = err
	if err == ErrWrongLeader{
		reply.WrongLeader = true
	}else{
		reply.WrongLeader = false
	}
	if err == OK {
		fmt.Println(sm.configs[len(sm.configs) - 1])
	}
}

func (sm *ShardMaster) Move(args *MoveArgs, reply *MoveReply) {
	// Your code here.
	op := Op{Opname:"Move", Shard:args.Shard, GID:args.GID, ClientId:args.ClientId, Seq:args.Seq}
	err, _ := sm.StartCommand(op)

	reply.Err = err
	if err == ErrWrongLeader{
		reply.WrongLeader = true
	}else{
		reply.WrongLeader = false
	}
}

func (sm *ShardMaster) Query(args *QueryArgs, reply *QueryReply) {
	// Your code here.
	//if args.Num == -1 || args.Num >= len(sm.configs){
	//	args.Num = len(sm.configs) - 1
	//}
	op := Op{Opname:"Query", Num: args.Num, ClientId:args.ClientId, Seq:args.Seq}
	//fmt.Println(sm.me, "op is ",op)
	err, conf := sm.StartCommand(op)

	reply.Config = conf
	reply.Err = err
	if err == ErrWrongLeader{
		reply.WrongLeader = true
	}else{
		reply.WrongLeader = false
	}
	//fmt.Println(reply.Config)
}

func LoadBalance(config *Config){
	num := len(config.Groups)
	grp := make([]int, 0)
	if num == 0 {
		return
	}
	for i := range config.Groups {
		grp = append(grp, i)
	}
	sort.Ints(grp)
	every := NShards / num
	mod := NShards % num
	var a int
	for _, k := range grp{
		a = k
		break
	}
	count := make(map[int]int)
	for i := range config.Shards{
		if config.Shards[i] == 0 {
			count[a] ++ 
			config.Shards[i] = a
		}else{
			count[config.Shards[i]]++
		}
	}
	for _, i := range grp{
		//fmt.Println(i,count[i])
		for count[i] < every{
			for j := range config.Shards{
				if count[config.Shards[j]] > every{
					count[i]++
					count[config.Shards[j]]--
					config.Shards[j] = i
					break
				}
			}
		}
	}
	//for i := range config.Shards{
	//	fmt.Println(i,config.Shards[i], count[config.Shards[i]])
	//}
	for _, i := range grp{
		if count[i] >= every + 1{
			mod--
		}
	}
	for _, i := range grp{
		if mod > 0 && count[i] < every + 1{
			for j := range config.Shards{
				if count[config.Shards[j]] > every + 1{
					count[i]++
					count[config.Shards[j]]--
					config.Shards[j] = i
					break
				}
			}
			mod--
		}
	}
}

// func (sm *ShardMaster) Apply(op Op){
	
// }

// func (sm *ShardMaster) Reply(op Op, index int){

// }

func (sm *ShardMaster) doApplyOp(){
	for{
		//Killed
		sm.mu.Lock()
		st := sm.state
		sm.mu.Unlock()
		if st == Killed {
			return
		}

		msg := <-sm.applyCh
		index := msg.CommandIndex
		if op, ok := msg.Command.(Op); ok{
			sm.mu.Lock()

			res := Result{}
			res.Opname = op.Opname
			res.ClientId = op.ClientId
			res.Seq = op.Seq
			res.config = sm.MakeEmptyConfig()
			//apply
			if sm.DupCheck(op.ClientId, op.Seq){
				newConfig := sm.MakeEmptyConfig()
				sm.CopyConfig(&newConfig, &sm.configs[len(sm.configs) - 1])
				newConfig.Num = sm.configs[len(sm.configs) - 1].Num + 1
			
				//fmt.Println(op.Opname)
				switch op.Opname{
				case "Join":
					for k, v := range op.Servers{
						newConfig.Groups[k] = make([]string, len(v))
						for i := 0; i < len(v); i++ {
							newConfig.Groups[k][i] = v[i]
						}
						//newConfig.Groups[k] = v
					}
					LoadBalance(&newConfig)
					//fmt.Println("finish join")
				case "Leave":
					for _, v := range op.GIDs{
						delete(newConfig.Groups, v)
						for j := range newConfig.Shards{
							if newConfig.Shards[j] == v{
								newConfig.Shards[j] = 0
							}
						}
					}
					LoadBalance(&newConfig)
				case "Move":
					newConfig.Shards[op.Shard] = op.GID
				}
				con := sm.MakeEmptyConfig()
				if op.Opname == "Query" {
					num := op.Num
					if op.Num == -1 || op.Num >= len(sm.configs) {
						num = len(sm.configs) - 1
					}
					sm.CopyConfig(&con, &sm.configs[num])
					fmt.Println(sm.me, "query num", num,", config is", con.Shards,", client", op.ClientId," seq", op.Seq)
					//con = sm.configs[num]
				} else {
					sm.configs = append(sm.configs, newConfig)
					fmt.Println(sm.me, "op", op.Opname,", new config is", newConfig.Shards)
				}
				sm.detectDup[op.ClientId] = Result{op.Opname, op.ClientId, op.Seq, con}
				
				sm.CopyConfig(&res.config, &con)
			}

			//reply
			ch, ok := sm.chanresult[index]

			if ok{
				select{
				case <- ch:
				default:
				}
				ch <- res
			}
			sm.mu.Unlock()
		}
	}
}

//
// the tester calls Kill() when a ShardMaster instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (sm *ShardMaster) Kill() {
	sm.rf.Kill()
	// Your code here, if desired.
	sm.mu.Lock()
	sm.state = Killed
	sm.mu.Unlock()
}

// needed by shardkv tester
func (sm *ShardMaster) Raft() *raft.Raft {
	return sm.rf
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Paxos to
// form the fault-tolerant shardmaster service.
// me is the index of the current server in servers[].
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister) *ShardMaster {
	sm := new(ShardMaster)
	sm.me = me
	sm.state = Alive
	sm.configs = make([]Config, 1)
	sm.configs[0] = sm.MakeEmptyConfig()

	labgob.Register(Op{})
	sm.detectDup = make(map[int64]Result)
	sm.chanresult = make(map[int]chan Result)

	sm.applyCh = make(chan raft.ApplyMsg)
	sm.rf = raft.Make(servers, me, persister, sm.applyCh)

	// Your code here.

	go sm.doApplyOp()
	return sm
}
