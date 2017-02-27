package orm

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MysqlDBConfig MySQL数据库
type MysqlDBConfig DBConfig

// NewDBPool 构建MySql数据库连接池
func (config *MysqlDBConfig) NewDBPool() (*DBPool, error) {
	if config == nil {
		return nil, &DBError{"Not found config", nil}
	}

	if len(config.User) == 0 || len(config.Url) == 0 || len(config.Schema) == 0 {
		return nil, &DBError{"Invalid config", nil}
	}

	//设置时间为本地时间,并解析时间
	loc, err := time.LoadLocation("Local")
	connectURL := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&loc=%s&parseTime=true", config.User, config.Pass, config.Url, config.Schema, url.QueryEscape(loc.String()))
	db, err := sql.Open("mysql", connectURL)
	if err != nil {
		log.Println("Error on initializing database connection,", err.Error())
		return nil, &DBError{"Can't open connection", err}
	}
	db.SetMaxIdleConns(config.MaxIdle)
	db.SetMaxOpenConns(config.MaxConn)
	return &DBPool{db}, nil
}

// MySQLDBService implements DBService interface for MySql
type MySQLDBService struct {
	Config *DBConfig `inject:"_"`
	pool   *DBPool
}

// Init implements Initable.Init()
func (p *MySQLDBService) Init() error {
	if p.pool != nil {
		return fmt.Errorf("Inited")
	}

	if p.Config == nil {
		return fmt.Errorf("No db config")
	}

	mysqlDbConfig := (*MysqlDBConfig)(p.Config)
	pool, err := mysqlDbConfig.NewDBPool()
	if err != nil {
		return err
	}
	p.pool = pool
	return nil
}

// NewDBOper implements DBService.NewDBOper()
func (p *MySQLDBService) NewDBOper() (*DBOper, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("please init db pool")
	}
	return p.pool.NewDBOper(), nil
}

// Pool implements DBService.Pool()
func (p *MySQLDBService) Pool() *DBPool {
	return p.pool
}
