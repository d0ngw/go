package cache

import (
	"fmt"
	"sort"
	"time"

	c "github.com/d0ngw/go/common"
	"github.com/gomodule/redigo/redis"
)

// Default Redis connection pool settings.
const (
	DefaultConnectTimout = 5 * 1000
	DefaultReadTimeout   = 5 * 1000
	DefaultWriteTimeout  = 5 * 1000
	DefaultMaxActive     = 100
	DefaultMaxIdle       = 2
	DefaultIdleTimeout   = 60 * 1000
)

// RedisConfigurer provides Redis configuration.
type RedisConfigurer interface {
	c.Configurer
	RedisConfig() *RedisConf
}

// RedisPoolConf configures a Redis connection pool.
type RedisPoolConf struct {
	ConnectTimeout int `yaml:"connect_timeout"` // Connection timeout in milliseconds.
	ReadTimeout    int `yaml:"read_timeout"`    // Read timeout in milliseconds.
	WriteTimeout   int `yaml:"write_timeout"`   // Write timeout in milliseconds.
	MaxIdle        int `yaml:"max_idle"`        // Maximum number of idle connections.
	MaxActive      int `yaml:"max_active"`      // Maximum number of active connections; zero means unlimited.
	IdleTimeout    int `yaml:"idel_teimout"`    // Idle connection timeout in milliseconds.
}

var defaultPool = &RedisPoolConf{
	ConnectTimeout: DefaultConnectTimout,
	ReadTimeout:    DefaultReadTimeout,
	WriteTimeout:   DefaultWriteTimeout,
	MaxActive:      DefaultMaxActive,
	MaxIdle:        DefaultMaxIdle,
	IdleTimeout:    DefaultIdleTimeout,
}

// RedisServer configures a Redis server.
type RedisServer struct {
	ID   string      `yaml:"id"`   // Redis server ID.
	Host string      `yaml:"host"` // Redis host address.
	Port int         `yaml:"port"` // Redis port.
	Auth string      `yaml:"auth"` // Redis authentication password.
	pool *redis.Pool // Redis connection pool.
}

// initPool initializes the pool with the supplied settings.
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
	Servers   []*RedisServer            `yaml:"servers"`      // Server configurations.
	Groups    map[string][]string       `yaml:"groups"`       // Group IDs mapped to lists of Redis server IDs.
	Pool      *RedisPoolConf            `yaml:"pool"`         // Default connection pool settings.
	GroupPool map[string]*RedisPoolConf `yaml:"groups_pools"` // Connection pool settings for each Redis group.
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

	// Parse and validate server configurations.
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

	// Parse and validate groups.
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

		// Sort Redis servers.
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
