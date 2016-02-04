//简单的Redis封装
package redis

import (
	"fmt"
	r "github.com/go-redis/redis"
	"github.com/yasushi-saito/rbtree"
)

//基于Slice实现的shard
//注:该实现不是线程安全的,如果同时添加client产生的后果是不可预知的

type sliceShard struct {
	nodeOpt *RedisNodeOpt
	c       *r.Client
}

type redisSliceShard struct {
	shards   []*sliceShard
	hashFunc HashFuncU32
}

//创建使用slice实现的RedisShard
func NewRedisSliceShard(nodes []RedisNodeOpt, hashFunc HashFuncU32) RedisShard {
	if hashFunc == nil {
		panic("No hashFunc")
	}
	checkDuplicateNode(nodes)
	s := &redisSliceShard{shards: make([]*sliceShard, 0), hashFunc: hashFunc}
	s.add(nodes)
	return s
}

func checkDuplicateNode(nodes []RedisNodeOpt) {
	nodeMap := make(map[string]bool)
	for _, node := range nodes {
		if nodeMap[node.Addr] {
			panic(fmt.Errorf("Duplicate node:%v", node))
		} else {
			nodeMap[node.Addr] = true
		}
	}
}

//添加Redis节点,如果重复添加相同的节点会报错
func (self *redisSliceShard) add(nodes []RedisNodeOpt) {
	for _, nodeOpt := range nodes {
		client := r.NewClient(nodeOpt.opt())
		//检测是否连通
		if nodeOpt.PingOnInit {
			status := client.Ping()
			if status.Err() != nil {
				panic(status.Err())
			}
		}
		shardOpt := nodeOpt
		shard := &sliceShard{
			nodeOpt: &shardOpt,
			c:       client}
		self.shards = append(self.shards, shard)
	}
}

func (self *redisSliceShard) Get(key []byte) *r.Client {
	count := uint32(len(self.shards))
	if count == 0 {
		return nil
	}
	hashKey := self.hashFunc(key)
	return self.shards[hashKey%count].c
}

func (self *redisSliceShard) GetS(key string) *r.Client {
	return self.Get([]byte(key))
}

//兼容Jedis Shard机制的实现
//注:该实现不是线程安全的,如果同时添加client产生的后果是不可预知的
type jedisShard struct {
	clients  []*r.Client
	tree     *rbtree.Tree
	hashFunc HashFuncI64
}

//jedis的TreeNode
type jedisTreeNode struct {
	key         int64 //hash key
	clientIndex int   //Redis Client Index
	nodeOpt     RedisNodeOpt
}

//构建Jedis Shard
func NewJedisShard(nodes []RedisNodeOpt) RedisShard {
	checkDuplicateNode(nodes)
	s := &jedisShard{
		clients: make([]*r.Client, 0),
		tree: rbtree.NewTree(func(a, b rbtree.Item) int {
			aNodeKey := a.(*jedisTreeNode).key
			bNodeKey := b.(*jedisTreeNode).key
			if aNodeKey == bNodeKey {
				return 0
			} else if aNodeKey < bNodeKey {
				return -1
			} else {
				return 1
			}
		}),
		hashFunc: func(data []byte) int64 {
			return MurmurHash64A_Jedis(data, 0x1234ABCD)
		}}
	s.add(nodes)
	return s
}

func (self *jedisShard) add(nodes []RedisNodeOpt) {
	for i, nodeOpt := range nodes {
		weight := nodeOpt.Weight
		if weight <= 0 {
			weight = 1
		}
		client := r.NewClient(nodeOpt.opt())
		//检测是否连通
		if nodeOpt.PingOnInit {
			status := client.Ping()
			if status.Err() != nil {
				panic(status.Err())
			}
		}
		self.clients = append(self.clients, client)
		for n := 0; n < int(160*weight); n++ {
			var name string
			if len(nodeOpt.ShardName) == 0 {
				name = fmt.Sprintf("SHARD-%d-NODE-%d", i, n)
			} else {
				name = fmt.Sprintf("%s*%d%d", nodeOpt.ShardName, weight, n)
			}
			hashKey := self.hashFunc([]byte(name))
			treeNode := &jedisTreeNode{key: hashKey, clientIndex: i, nodeOpt: nodeOpt}
			self.tree.Insert(treeNode)
		}
	}
}

func (self *jedisShard) get(key []byte) *jedisTreeNode {
	hashKey := self.hashFunc(key)
	itr := self.tree.FindGE(&jedisTreeNode{key: hashKey})
	if !itr.Limit() {
		return itr.Item().(*jedisTreeNode)
	} else {
		return self.tree.Min().Item().(*jedisTreeNode)
	}
}

func (self *jedisShard) Get(key []byte) *r.Client {
	return self.clients[self.get(key).clientIndex]
}

func (self *jedisShard) GetS(key string) *r.Client {
	return self.Get([]byte(key))
}
