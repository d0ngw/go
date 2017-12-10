package orm

import (
	c "github.com/d0ngw/go/common"
)

// DBConfigurer DB配置器
type DBConfigurer interface {
	c.Configurer
	// DBConfig 取得数据库的配置
	DBConfig() *DBConfig
}
