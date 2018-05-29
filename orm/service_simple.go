package orm

import (
	"fmt"

	c "github.com/d0ngw/go/common"
)

// DBService is the service that supply DBOper
type DBService interface {
	c.Initable
	OpCreator
}

// SimpleDBService implements DBService interface
type SimpleDBService struct {
	Config   DBConfigurer `inject:"_"`
	poolFunc PoolFunc
	pool     *Pool
}

// NewSimpleDBService build simple db service
func NewSimpleDBService(poolFunc PoolFunc) *SimpleDBService {
	return &SimpleDBService{poolFunc: poolFunc}
}

// Init implements Initable.Init()
func (p *SimpleDBService) Init() error {
	if p.poolFunc == nil {
		return fmt.Errorf("no pool func")
	}
	if p.pool != nil {
		return fmt.Errorf("Inited")
	}
	if p.Config == nil {
		return fmt.Errorf("No db config")
	}

	pool, err := p.poolFunc(p.Config.DBConfig())
	if err != nil {
		return err
	}
	p.pool = pool
	return nil
}

// NewOp implements DBService.NewOp()
func (p *SimpleDBService) NewOp() (*Op, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("please init db pool")
	}
	return p.pool.NewOp(), nil
}
