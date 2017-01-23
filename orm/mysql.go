package orm

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/url"
	"time"
)

//MySQL数据库
type MysqlDBConfig DBConfig

//CreateDBPool 构建MySql数据库连接池
func (config *MysqlDBConfig) NewDBPool() (*DBPool, error) {
	if config == nil {
		return nil, &DBError{"Not found config", nil}
	}

	if len(config.User) == 0 || len(config.Url) == 0 || len(config.Schema) == 0 {
		return nil, &DBError{"Invalid config", nil}
	}

	//设置时间为本地时间,并解析时间
	loc, err := time.LoadLocation("Local")
	connect_url := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&loc=%s&parseTime=true", config.User, config.Pass, config.Url, config.Schema, url.QueryEscape(loc.String()))
	db, err := sql.Open("mysql", connect_url)
	if err != nil {
		log.Println("Error on initializing database connection,", err.Error())
		return nil, &DBError{"Can't open connection", err}
	}
	db.SetMaxIdleConns(config.MaxIdle)
	db.SetMaxOpenConns(config.MaxConn)
	return &DBPool{db}, nil
}
