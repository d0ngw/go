package orm

import (
	"fmt"

	c "github.com/d0ngw/go/common"
)

// DBConfigurer DB配置器
type DBConfigurer interface {
	c.Configurer
	DBConfig() *DBConfig
}

// DBShardConfigurer db shard configurer
type DBShardConfigurer interface {
	c.Configurer
	DBShardConfig() *DBShardConfig
}

// EntityShardConfigurer entity shard configurer
type EntityShardConfigurer interface {
	c.Configurer
	EntityShardConfig() *EntityShardConfig
}

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

// DBShardConfig implements DBShardConfigurer
func (p *DBShardConfig) DBShardConfig() *DBShardConfig {
	return p
}

// ShardRuleConfig shard规则配置
type ShardRuleConfig struct {
	Name    string   `yaml:"name"`
	OneRule *OneRule `yaml:"one"`
}

// EntityShardConfig entity shad config
type EntityShardConfig struct {
	// pkgPath -> entity name -> shard name
	Shards map[string]map[string]*ShardRuleConfig `yaml:"shards"`
}

// Parse implements Configurer.Parse
func (p *EntityShardConfig) Parse() error {
	return nil
}

// EntityShardConfig implements EntityShardConfigurer
func (p *EntityShardConfig) EntityShardConfig() *EntityShardConfig {
	return p
}
