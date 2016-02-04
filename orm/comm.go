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
	DB *sql.DB
}

//数据库配置
type DBConfig struct {
	USER     string
	PASS     string
	URL      string
	SCHEMA   string
	MAX_CONN int
	MAX_IDLE int
}

//数据库连接池创建
type DBPoolCreater interface {
	CreateDBPool(config DBConfig) (*DBPool, error)
}
