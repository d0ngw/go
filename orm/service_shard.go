package orm

import (
	"fmt"

	c "github.com/d0ngw/go/common"
)

// ShardDBService implements DBService interface
type ShardDBService struct {
	DBConfig     DBShardConfigurer     `inject:"_"`
	EntityConfig EntityShardConfigurer `inject:"_,optional"`
	poolFunc     PoolFunc
	pools        map[string]*Pool
	defaultPool  *Pool
}

// Init implements Initable.Init()
func (p *ShardDBService) Init() error {
	if p.poolFunc == nil {
		return fmt.Errorf("no pool func")
	}
	if p.pools != nil {
		return fmt.Errorf("Inited")
	}
	if p.DBConfig == nil {
		return fmt.Errorf("no db shard config")
	}

	pools := map[string]*Pool{}
	var defaultPool *Pool

	config := p.DBConfig.DBShardConfig()
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

// NewOpByMeta create Op by meta
func (p *ShardDBService) NewOpByMeta(meta Meta) (op *Op, err error) {
	if meta == nil {
		err = fmt.Errorf("invalid meta")
		return
	}
	return
}

func (p *ShardDBService) SetUpShardFunc(shardEntity ShardEntity, val interface{}) func() string {

	return nil
}
