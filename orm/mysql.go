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

//构建MySql数据的
func (config *MysqlDBConfig) CreateDBPool() (*DBPool, error) {
	if config == nil {
		return nil, &DBError{"Not found config", nil}
	}

	if len(config.USER) == 0 || len(config.URL) == 0 || len(config.SCHEMA) == 0 {
		return nil, &DBError{"Invalid config", nil}
	}

	//设置时间为本地时间,并解析时间
	loc, err := time.LoadLocation("Local")
	connect_url := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&loc=%s&parseTime=true", config.USER, config.PASS, config.URL, config.SCHEMA, url.QueryEscape(loc.String()))
	db, err := sql.Open("mysql", connect_url)
	if err != nil {
		log.Println("Error on initializing database connection,", err.Error())
		return nil, &DBError{"Can't open connection", err}
	}
	db.SetMaxIdleConns(config.MAX_IDLE)
	db.SetMaxOpenConns(config.MAX_CONN)
	return &DBPool{db}, nil
}
