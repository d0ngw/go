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
	c.Infof("db shards count:%d", len(p.Shards))

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

// EntityShardRuleConfig 实体的shard规则
type EntityShardRuleConfig struct {
	Name       string   `yaml:"name"`        //名称
	DBShard    *OneRule `yaml:"db_shard"`    //数据库实例的配置
	TableShard *OneRule `yaml:"table_shard"` //数据库表的配置
	Default    bool     `yaml:"default"`     //是否是默认规则
	meta       Meta
}

// Parse implements Configurer.Parse
func (p *EntityShardRuleConfig) Parse() error {
	if p.Name == "" {
		return fmt.Errorf("invalid name")
	}
	if p.DBShard != nil {
		if err := p.DBShard.Parse(); err != nil {
			return fmt.Errorf("parse db_shard fail,name:%s,err:%v", p.Name, err)
		}
	}
	if p.TableShard != nil {
		if err := p.TableShard.Parse(); err != nil {
			return fmt.Errorf("parse table_shard fail,name:%s,err:%v", p.Name, err)
		}
	}
	return nil
}

type entityRule struct {
	meta        Meta
	defaultRule *EntityShardRuleConfig
	rules       map[string]*EntityShardRuleConfig
}

// EntityShardConfig entity shad config
type EntityShardConfig struct {
	// pkgPath -> entity name -> rules
	Entities map[string]map[string][]*EntityShardRuleConfig `yaml:"entities"`
	entities map[string]map[string]*entityRule
}

// Parse implements Configurer.Parse
func (p *EntityShardConfig) Parse() error {
	if len(p.Entities) == 0 {
		c.Infof("no entities")
		return nil
	}

	entities := map[string]map[string]*entityRule{}

	for pkgPath, pkgEntities := range p.Entities {
		pkg := map[string]*entityRule{}
		entities[pkgPath] = pkg
		for entityName, rules := range pkgEntities {
			meta := findMetaWithPkgAndName(pkgPath, entityName)
			if meta == nil {
				return fmt.Errorf("can't find meta for %s.%s", pkgPath, entityName)
			}

			entity := &entityRule{rules: map[string]*EntityShardRuleConfig{}, meta: meta}
			pkg[entityName] = entity

			for _, rule := range rules {
				if err := rule.Parse(); err != nil {
					return fmt.Errorf("parse %s/%s %s fail,err:%v", pkgPath, entityName, rule.Name, err)
				}
				if rule.Default {
					if entity.defaultRule == nil {
						entity.defaultRule = rule
					} else {
						return fmt.Errorf("duplicate default rule for %s/%s", pkgPath, entityName)
					}
				}
				if entity.rules[rule.Name] != nil {
					return fmt.Errorf("duplicate default rule for %s/%s", pkgPath, entityName)
				}
				entity.rules[rule.Name] = rule
			}
		}
	}
	p.entities = entities
	return nil
}

// EntityShardConfig implements EntityShardConfigurer
func (p *EntityShardConfig) EntityShardConfig() *EntityShardConfig {
	return p
}
