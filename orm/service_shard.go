package orm

import (
	"fmt"

	c "github.com/d0ngw/go/common"
)

// ShardDBService implements DBService interface
type ShardDBService struct {
	DBShardConfig     DBShardConfigurer     `inject:"_"`
	EntityShardConfig EntityShardConfigurer `inject:"_,optional"`
	poolFunc          PoolFunc
	pools             map[string]*Pool
	defaultPool       *Pool
}

// Init implements Initable.Init()
func (p *ShardDBService) Init() error {
	if p.poolFunc == nil {
		return fmt.Errorf("no pool func")
	}
	if p.pools != nil {
		return fmt.Errorf("Inited")
	}
	if p.DBShardConfig == nil {
		return fmt.Errorf("no db shard config")
	}

	pools := map[string]*Pool{}
	var defaultPool *Pool

	config := p.DBShardConfig.DBShardConfig()
	if config == nil {
		c.Warnf("no db shard config")
	} else {
		for k, v := range config.Shards {
			pool, err := p.poolFunc(v)
			if err != nil {
				return err
			}
			pools[k] = pool
		}

		if config.Default != "" {
			defaultPool = pools[config.Default]
			if defaultPool == nil {
				return fmt.Errorf("can't find default pool for %s", config.Default)
			}
		}
		p.pools = pools
		p.defaultPool = defaultPool
		if p.defaultPool == nil {
			return fmt.Errorf("no default pool")
		}
	}
	return nil
}

// NewOp create default op
func (p *ShardDBService) NewOp() *Op {
	return p.defaultPool.NewOp()
}

// NewOpByName create Op by pool name
func (p *ShardDBService) NewOpByName(poolName string) (op *Op, err error) {
	pool := p.pools[poolName]
	if pool == nil {
		err = fmt.Errorf("can't find pool by name %s", poolName)
		return
	}
	op = pool.NewOp()
	return
}

// NewOpByEntity create Op for entity with default rule
func (p *ShardDBService) NewOpByEntity(entity Entity) (op *Op, err error) {
	return p.newOpByEntity(entity, "")
}

// NewOpByEntityWithRuleName create op for entity with shard rule name
func (p *ShardDBService) NewOpByEntityWithRuleName(entity Entity, ruleName string) (op *Op, err error) {
	return p.newOpByEntity(entity, ruleName)
}

func (p *ShardDBService) newOpByEntity(entity Entity, ruleName string) (op *Op, err error) {
	if entity == nil {
		err = fmt.Errorf("invalid meta")
		return
	}

	rule, err := p.findShardRule(entity, ruleName)
	if err != nil {
		return
	}

	pool, err := p.findShardPool(entity, rule)
	if err != nil {
		return
	}

	if shardEntity, ok := entity.(ShardEntity); ok {
		handler, err := p.findTableShardHandler(entity, rule)
		if err != nil {
			return nil, err
		}
		if handler != nil {
			shardEntity.SetTableShardFunc(handler)

		}
	}

	op = pool.NewOp()
	return
}

func (p *ShardDBService) findShardRule(entity Entity, ruleName string) (rule *EntityShardRuleConfig, err error) {
	if p.EntityShardConfig == nil || p.EntityShardConfig.EntityShardConfig() == nil {
		return nil, nil
	}
	_, _, typ := extract(entity)

	pkgPath := typ.PkgPath()
	name := typ.Name()

	shardConf := p.EntityShardConfig.EntityShardConfig()

	pkg := shardConf.entities[pkgPath]
	if pkg == nil {
		return nil, nil
	}

	entityRules := pkg[name]
	if entityRules == nil {
		return nil, nil
	}

	if ruleName != "" {
		rule = entityRules.rules[ruleName]
		if rule == nil {
			return nil, fmt.Errorf("can't find rule for %s.%s for rule %s", pkgPath, name, ruleName)
		}
	} else {
		rule = entityRules.defaultRule
	}
	return
}

func (p *ShardDBService) findShardPool(entity Entity, rule *EntityShardRuleConfig) (pool *Pool, err error) {
	if rule == nil || rule.DBShard == nil {
		return p.getDefaultPool()
	}

	dbShardRule := rule.DBShard

	fieldName := dbShardRule.ShardFieldName()
	var fieldVal interface{}

	if fieldName != "" {
		fieldVal, err = rule.meta.FieldValue(entity, fieldName)
		if err != nil {
			return
		}
	}

	shardName, err := dbShardRule.Shard(fieldVal)
	if err != nil {
		return
	}

	pool = p.pools[shardName]
	if pool == nil {
		err = fmt.Errorf("can't find pool by name %s", shardName)
		return
	}
	return
}

func (p *ShardDBService) findTableShardHandler(entity Entity, rule *EntityShardRuleConfig) (handler ShardHandler, err error) {
	if rule == nil || rule.TableShard == nil {
		return
	}

	tableShardRule := rule.TableShard

	fieldName := tableShardRule.ShardFieldName()
	var fieldVal interface{}

	if fieldName != "" {
		fieldVal, err = rule.meta.FieldValue(entity, fieldName)
		if err != nil {
			return
		}
	}

	shardName, err := tableShardRule.Shard(fieldVal)
	if err != nil {
		return
	}

	handler = func() (string, error) {
		return shardName, nil
	}
	return
}

func (p *ShardDBService) getDefaultPool() (pool *Pool, err error) {
	if p.defaultPool != nil {
		return p.defaultPool, nil
	}
	return nil, fmt.Errorf("no default pool")
}
