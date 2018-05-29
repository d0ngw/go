package orm

import (
	"fmt"

	c "github.com/d0ngw/go/common"
)

//DBConfig 数据库配置
type DBConfig struct {
	User          string `yaml:"user"`
	Pass          string `yaml:"pass"`
	URL           string `yaml:"url"`
	Schema        string `yaml:"schema"`
	MaxConn       int    `yaml:"maxConn"`
	MaxIdle       int    `yaml:"maxIdle"`
	MaxTimeSecond int    `yaml:"maxTimeSecond"`
	Charset       string `yaml:"charset"`
}

// Parse implements DBConfigurer
func (p *DBConfig) Parse() error {
	if p.URL == "" {
		return fmt.Errorf("need url")
	}
	if p.Schema == "" {
		return fmt.Errorf("need schema")
	}
	return nil
}

// DBConfig implements DBConfigurer
func (p *DBConfig) DBConfig() *DBConfig {
	return p
}

// DBConfigurer DB配置器
type DBConfigurer interface {
	c.Configurer
	DBConfig() *DBConfig
}

// DBShardConfig db shard config
type DBShardConfig struct {
	Shards  map[string]*DBConfig `yaml:"shards"`
	Default string               `yaml:"default"`
}

// Parse implements Configurer.Parse
func (p *DBShardConfig) Parse() error {
	for k, v := range p.Shards {
		if v == nil {
			return fmt.Errorf("no db config for %s", k)
		}
		if err := v.Parse(); err != nil {
			return err
		}
	}
	if p.Default != "" {
		if p.Shards[p.Default] == nil {
			return fmt.Errorf("can't find default shard %s", p.Default)
		}
	}
	return nil
}

// ShardConfig implements DBShardConfigurer
func (p *DBShardConfig) ShardConfig() *DBShardConfig {
	return p
}

// DBShardConfigurer db shard configurer
type DBShardConfigurer interface {
	c.Configurer
	ShardConfig() *DBShardConfig
}
