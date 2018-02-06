package orm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"

	c "github.com/d0ngw/go/common"
	"github.com/go-sql-driver/mysql"
)

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

// EntityInterface  模型的基本接口
type EntityInterface interface {
	TableName() string
}

// SliceEntityInterface type for slice of EntityInterface
type SliceEntityInterface []EntityInterface

// InterfaceSlice convert SliceEntityInterface to []interface{}
func (p SliceEntityInterface) InterfaceSlice() []interface{} {
	if p == nil {
		return nil
	}
	ret := make([]interface{}, len(p))
	for i := range p {
		ret[i] = p[i]
	}
	return ret
}

// DBPool 数据库连接池
type DBPool struct {
	db *sql.DB
}

//NewDBOper 创建DBOper
func (p *DBPool) NewDBOper() *DBOper {
	return &DBOper{db: p.db}
}

//DBConfig 数据库配置
type DBConfig struct {
	User          string `yaml:"user"`
	Pass          string `yaml:"pass"`
	Url           string `yaml:"url"`
	Schema        string `yaml:"schema"`
	MaxConn       int    `yaml:"maxConn"`
	MaxIdle       int    `yaml:"maxIdle"`
	MaxTimeSecond int    `yaml:"maxTimeSecond"`
	Charset       string `yaml:"charset"`
}

// DBPoolCreator 数据库连接池创建
type DBPoolCreator interface {
	//NewDBPool 创建数据库连接池
	NewDBPool(config DBConfig) (*DBPool, error)
}

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

// DBService is the service that supply DBOper
type DBService interface {
	c.Initable
	//NewDBOper create a new DBOper
	NewDBOper() (*DBOper, error)
	// Pool get the db pool
	Pool() *DBPool
}
