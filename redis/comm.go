package redis

import (
	r "github.com/go-redis/redis" //TODO 等新版本发布后,使用gopkg.in包
	"time"
)

//Redis节点配置
type RedisNodeOpt struct {
	//Shard的名称
	ShardName string
	//节点的权重
	Weight uint
	//是否在初始化的时候ping
	PingOnInit bool
	//Shard的地址
	Addr string
	//Redis密码
	Password string
	//Redis DB, 默认: 0
	DB int64
	//链接超时,默认5秒
	DialTimeout time.Duration
	//读超时,默认5秒
	ReadTimeout time.Duration
	//写超时,默认5秒
	WriteTimeout time.Duration
	//连接池大小,默认10
	PoolSize int
	//连接池超时时间 5秒
	PoolTimeout time.Duration
	//连接空闲时间,默认0(不淘汰)
	IdleTimeout time.Duration
}

func (self *RedisNodeOpt) opt() *r.Options {
	return &r.Options{
		Addr:         self.Addr,
		Password:     self.Password,
		DB:           self.DB,
		DialTimeout:  self.DialTimeout,
		ReadTimeout:  self.ReadTimeout,
		WriteTimeout: self.WriteTimeout,
		PoolSize:     self.PoolSize,
		PoolTimeout:  self.PoolTimeout,
		IdleTimeout:  self.IdleTimeout}
}

//Redis客户端Shard接口
type RedisShard interface {
	//根据Key取得RedisClient
	Get(key []byte) *r.Client
	//根据Key取得RedisClient
	GetS(key string) *r.Client
}
