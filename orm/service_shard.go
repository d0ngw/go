package orm

import (
	"fmt"

	c "github.com/d0ngw/go/common"
)

// ShardDBService 支持分库,分表的DBService
type ShardDBService interface {
	DBService
	// NewOpByShardName create op by shard name
	NewOpByShardName(name string) (op *Op, err error)
	// NewOpByEntity create Op for entity with rule name,if rule name is empty use default rule
	NewOpByEntity(entity Entity, ruleName string) (op *Op, err error)
	// setupTableShard setup ShardEntity.TableShardFunc with rule name,if rule name is empty use default rule
	setupTableShard(entity Entity, ruleName string) (poolName string, err error)
}

// SimpleShardDBService implements DBService interface
type SimpleShardDBService struct {
	DBShardConfig     DBShardConfigurer     `inject:"_"`
	EntityShardConfig EntityShardConfigurer `inject:"_,optional"`
	poolFunc          PoolFunc
	pools             map[string]*Pool
	defaultPool       *Pool
}

// NewSimpleShardDBService create
func NewSimpleShardDBService(poolFunc PoolFunc) *SimpleShardDBService {
	return &SimpleShardDBService{poolFunc: poolFunc}
}

// Init implements Initable.Init()
func (p *SimpleShardDBService) Init() error {
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
			pool.name = k
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
func (p *SimpleShardDBService) NewOp() (op *Op, err error) {
	pool, err := p.getDefaultPool()
	if err != nil {
		return
	}
	op = pool.NewOp()
	op.sharDBSerevcie = p
	return
}

// NewOpByShardName create Op by shard name
func (p *SimpleShardDBService) NewOpByShardName(poolName string) (op *Op, err error) {
	pool := p.pools[poolName]
	if pool == nil {
		err = fmt.Errorf("can't find pool by name %s", poolName)
		return
	}
	op = pool.NewOp()
	op.sharDBSerevcie = p
	return
}

// NewOpByEntity create Op for entity with rule name,if rule name is empty use default rule
func (p *SimpleShardDBService) NewOpByEntity(entity Entity, ruleName string) (op *Op, err error) {
	pool, err := p.matchPoolAndSetupTblShard(entity, ruleName)
	if err != nil {
		return
	}

	op = pool.NewOp()
	op.sharDBSerevcie = p
	return
}

// setupTableShard setup ShardEntity.TableShardFunc
func (p *SimpleShardDBService) setupTableShard(entity Entity, ruleName string) (poolName string, err error) {
	pool, err := p.matchPoolAndSetupTblShard(entity, ruleName)
	if err != nil {
		return
	}
	poolName = pool.Name()
	return
}

func (p *SimpleShardDBService) findShardRule(entity Entity, ruleName string) (rule *EntityShardRuleConfig, err error) {
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

func (p *SimpleShardDBService) findShardPool(entity Entity, rule *EntityShardRuleConfig) (pool *Pool, err error) {
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

func (p *SimpleShardDBService) findTableShardHandler(entity Entity, rule *EntityShardRuleConfig) (handler ShardHandler, err error) {
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

func (p *SimpleShardDBService) matchPoolAndSetupTblShard(entity Entity, ruleName string) (pool *Pool, err error) {
	if entity == nil {
		err = fmt.Errorf("invalid meta")
		return
	}

	rule, err := p.findShardRule(entity, ruleName)
	if err != nil {
		return
	}

	pool, err = p.findShardPool(entity, rule)
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
	return
}

func (p *SimpleShardDBService) getDefaultPool() (pool *Pool, err error) {
	if p.defaultPool != nil {
		return p.defaultPool, nil
	}
	return nil, fmt.Errorf("no default pool")
}
