package orm

import (
	"database/sql"
	"fmt"

	// import mysql
	_ "github.com/go-sql-driver/mysql"
)

// NullTime null time
type NullTime = sql.NullTime

// DBError describes a database operation error.
type DBError struct {
	Msg string
	Err error
}

func (e *DBError) Error() string {
	return fmt.Sprintf("DBError msg:%s,err:%v", e.Msg, e.Err)
}

// NewDBError creates a database operation error.
func NewDBError(err error, msg string) *DBError {
	return &DBError{Msg: msg, Err: err}
}

// NewDBErrorf creates a formatted database operation error.
func NewDBErrorf(err error, msgFormat string, args ...interface{}) *DBError {
	return &DBError{Msg: fmt.Sprintf(msgFormat, args...), Err: err}
}

// Entity identifies a database entity.
type Entity interface {
	TableName() string
}

// ShardHandler determines the shard for an entity.
type ShardHandler func() (shardName string, err error)

// ShardEntity defines an entity that supports table sharding.
type ShardEntity interface {
	Entity
	// TableShardFunc returns the table sharding function.
	TableShardFunc() ShardHandler
	// SetTableShardFunc sets the table sharding function.
	SetTableShardFunc(ShardHandler)
}

// BaseShardEntity provides table sharding support.
type BaseShardEntity struct {
	tblShardFunc ShardHandler
}

// TableShardFunc implements ShardEntity.TableShardFunc
func (p *BaseShardEntity) TableShardFunc() ShardHandler {
	return p.tblShardFunc
}

// SetTableShardFunc implements ShardEntity.SetTableShardFunc
func (p *BaseShardEntity) SetTableShardFunc(f ShardHandler) {
	p.tblShardFunc = f
}

// EntitySlice type for slice of EntityInterface
type EntitySlice []Entity

// ToInterface convert EntitySlice to []interface{}
func (p EntitySlice) ToInterface() []interface{} {
	if p == nil {
		return nil
	}
	ret := make([]interface{}, len(p))
	for i := range p {
		ret[i] = p[i]
	}
	return ret
}

// Pool wraps a database connection pool.
type Pool struct {
	db   *sql.DB
	name string
}

// Close releases the database pool and its connections.
func (p *Pool) Close() error {
	return p.db.Close()
}

// NewOp creates a database operation.
func (p *Pool) NewOp() *Op {
	return &Op{pool: p}
}

// Name pool name
func (p *Pool) Name() string {
	return p.name
}

// PoolFunc the func to crate db pool
type PoolFunc func(config *DBConfig) (pool *Pool, err error)
