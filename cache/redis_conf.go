package cache

import (
	"fmt"
	"sort"
	"time"

	c "github.com/d0ngw/go/common"
	"github.com/gomodule/redigo/redis"
)

// Redis连接池的默认参数
const (
	DefaultConnectTimout = 5 * 1000
	DefaultReadTimeout   = 5 * 1000
	DefaultWriteTimeout  = 5 * 1000
	DefaultMaxActive     = 100
	DefaultMaxIdle       = 2
	DefaultIdleTimeout   = 60 * 1000
)

// RedisConfigurer Redis配置器
type RedisConfigurer interface {
	c.Configurer
	RedisConfig() *RedisConf
}

// RedisPoolConf  Redis连接池配置
type RedisPoolConf struct {
	ConnectTimeout int `yaml:"connect_timeout"` //连接超时时间,单位毫秒秒
	ReadTimeout    int `yaml:"read_timeout"`    //读取超时,单位毫秒
	WriteTimeout   int `yaml:"write_timeout"`   //写取超时,单位毫秒
	MaxIdle        int `yaml:"max_idle"`        //最大空闲连接
	MaxActive      int `yaml:"max_active"`      //最大活跃连接,0表示不限制
	IdleTimeout    int `yaml:"idel_teimout"`    //空闲连接的超时时间,单位毫秒
}

var defaultPool = &RedisPoolConf{
	ConnectTimeout: DefaultConnectTimout,
	ReadTimeout:    DefaultReadTimeout,
	WriteTimeout:   DefaultWriteTimeout,
	MaxActive:      DefaultMaxActive,
	MaxIdle:        DefaultMaxIdle,
	IdleTimeout:    DefaultIdleTimeout,
}

// RedisServer Redis实例的配置
type RedisServer struct {
	ID   string      `yaml:"id"`   //Redis实例的id
	Host string      `yaml:"host"` //Redis主机地址
	Port int         `yaml:"port"` //Redis的端口
	Auth string      `yaml:"auth"` //Redis认证密码
	pool *redis.Pool //Redis实例的连接池
}

// initPool 使用指定的参数初始化pool
func (p *RedisServer) initPool(poolConf *RedisPoolConf) error {
	if p.pool != nil {
		return fmt.Errorf("server %s already inited", p.ID)
	}
	options := []redis.DialOption{}
	options = append(options, redis.DialConnectTimeout(time.Duration(poolConf.ConnectTimeout)*time.Millisecond))
	options = append(options, redis.DialReadTimeout(time.Duration(poolConf.ReadTimeout)*time.Millisecond))
	options = append(options, redis.DialWriteTimeout(time.Duration(poolConf.WriteTimeout)*time.Millisecond))
	if p.Auth != "" {
		options = append(options, redis.DialPassword(p.Auth))
	}

	var addr = fmt.Sprintf("%s:%d", p.Host, p.Port)

	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr, options...)
		},
		MaxActive:   poolConf.MaxActive,
		MaxIdle:     poolConf.MaxIdle,
		IdleTimeout: time.Duration(poolConf.IdleTimeout) * time.Millisecond,
		Wait:        true,
	}
	p.pool = pool
	return nil
}

// GetConn acquire redis conn
func (p *RedisServer) GetConn() (redis.Conn, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("no pool")
	}
	return p.pool.Get(), nil
}

// RedisConf redis config
type RedisConf struct {
	Servers   []*RedisServer            `yaml:"servers"`      //实例列表
	Groups    map[string][]string       `yaml:"groups"`       //Redis组定义,key为组ID;value为Server的id列表
	Pool      *RedisPoolConf            `yaml:"pool"`         //默认的链接池配置
	GroupPool map[string]*RedisPoolConf `yaml:"groups_pools"` //Redis组的连接池配置
	groups    map[string][]*RedisServer
}

// Parse implements Configurer interface
func (p *RedisConf) Parse() error {
	if p == nil {
		c.Warnf("no redis conf")
		return nil
	}
	groups := map[string][]*RedisServer{}
	servers := map[string]*RedisServer{}

	//解析,并检查server的配置
	var dupChekc = map[string]struct{}{}
	for _, server := range p.Servers {
		if c.IsEmpty(server.ID, server.Host) {
			return fmt.Errorf("invalid redis server conf,id and host must not be emtpy")
		}
		if server.Port <= 0 {
			return fmt.Errorf("invalid redis server conf,port %d ", server.Port)
		}

		id := "id " + server.ID
		if _, ok := dupChekc[id]; ok {
			return fmt.Errorf("duplicate server:%s", id)
		}
		dupChekc[id] = struct{}{}

		addr := fmt.Sprintf("%s:%d", server.Host, server.Port)
		if _, ok := dupChekc[addr]; ok {
			return fmt.Errorf("duplicate server: %s", addr)
		}
		dupChekc[addr] = struct{}{}
		servers[server.ID] = server
	}

	//解析并检查group
	for groupID, groupServers := range p.Groups {
		if groupID == "" {
			return fmt.Errorf("invalid redis group id")
		}
		if len(servers) == 0 {
			return fmt.Errorf("redis group id %s has no servers", groupID)
		}
		dupChekc = map[string]struct{}{}
		for _, serverID := range groupServers {
			if _, ok := dupChekc[serverID]; ok {
				return fmt.Errorf("duplicate server id %s in group  %s", serverID, groupID)
			}
			dupChekc[serverID] = struct{}{}
		}

		poolConf := p.GroupPool[groupID]
		if poolConf == nil {
			poolConf = p.Pool
		}
		if poolConf == nil {
			poolConf = defaultPool
		}

		//对redis实例进行排序
		sort.Sort(sort.StringSlice(groupServers))
		redisServers := make([]*RedisServer, 0, len(groupServers))
		for _, serverID := range groupServers {
			server := servers[serverID]
			if server == nil {
				return fmt.Errorf("can't find server id %s", serverID)
			}
			groupServer := *server
			if err := groupServer.initPool(poolConf); err != nil {
				return err
			}
			redisServers = append(redisServers, &groupServer)
		}
		groups[groupID] = redisServers
	}
	p.groups = groups
	return nil
}

// RedisConfig implements RedisConfigurer
func (p *RedisConf) RedisConfig() *RedisConf {
	return p
}
