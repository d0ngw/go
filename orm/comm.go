package orm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/go-sql-driver/mysql"
)

// NullTime null time
type NullTime mysql.NullTime

// Scan null time scan
func (nt *NullTime) Scan(value interface{}) (err error) {
	return (*mysql.NullTime)(nt).Scan(value)
}

// Value null time value
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

// DBError 数据库操作错误
type DBError struct {
	Msg string
	Err error
}

func (e *DBError) Error() string {
	return fmt.Sprintf("DBError msg:%s,err:%v", e.Msg, e.Err)
}

// NewDBError  构建数据库操作错误
func NewDBError(err error, msg string) *DBError {
	return &DBError{Msg: msg, Err: err}
}

// NewDBErrorf  使用fmt.Sprintf构建
func NewDBErrorf(err error, msgFormat string, args ...interface{}) *DBError {
	return &DBError{Msg: fmt.Sprintf(msgFormat, args...), Err: err}
}

// Entity  实体接口
type Entity interface {
	TableName() string
}

// ShardHandler 分片处理
type ShardHandler func() (shardName string, err error)

// ShardEntity 支持按表分片实体的接口
type ShardEntity interface {
	Entity
	//TableShardFunc table分片函数
	TableShardFunc() ShardHandler
	//SetTableShardFunc table设置分片函数
	SetTableShardFunc(ShardHandler)
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

// Pool 数据库连接池
type Pool struct {
	db *sql.DB
}

//NewOp 创建DBOper
func (p *Pool) NewOp() *Op {
	return &Op{pool: p}
}

// PoolFunc the func to crate db pool
type PoolFunc func(config *DBConfig) (pool *Pool, err error)
