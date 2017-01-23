package orm

import (
	"database/sql"
	"fmt"
)

//数据库操作错误
type DBError struct {
	Msg string
	Err error
}

func (e *DBError) Error() string {
	return fmt.Sprintf("DBError msg:%s,err:%v", e.Msg, e.Err)
}

//构建数据库操作错误
func NewDBError(err error, msg string) *DBError {
	return &DBError{Msg: msg, Err: err}
}

//使用fmt.Sprintf构建
func NewDBErrorf(err error, msgFormat string, args ...interface{}) *DBError {
	return &DBError{Msg: fmt.Sprintf(msgFormat, args...), Err: err}
}

//模型的基本接口
type EntityInterface interface {
	TableName() string
}

//数据库连接池
type DBPool struct {
	db *sql.DB
}

//NewDBOper 创建DBOper
func (p *DBPool) NewDBOper() *DBOper {
	return &DBOper{db: p.db}
}

//DBConfig 数据库配置
type DBConfig struct {
	User    string
	Pass    string
	Url     string
	Schema  string
	MaxConn int
	MaxIdle int
}

// DBPoolCreator 数据库连接池创建
type DBPoolCreator interface {
	//NewDBPool 创建数据库连接池
	NewDBPool(config DBConfig) (*DBPool, error)
}
