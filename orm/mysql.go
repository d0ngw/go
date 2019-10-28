package orm

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"time"

	c "github.com/d0ngw/go/common"
	//init mysql
)

// NewMySQLDBPool build mysql db pool from config
func NewMySQLDBPool(config *DBConfig) (*Pool, error) {
	if config == nil {
		return nil, &DBError{"Not found config", nil}
	}

	if len(config.User) == 0 || len(config.URL) == 0 || len(config.Schema) == 0 {
		return nil, &DBError{"Invalid config", nil}
	}

	charset := config.Charset
	if charset == "" {
		charset = "utf8"
	}

	//设置时间为本地时间,并解析时间
	loc, err := time.LoadLocation("Local")
	connectURL := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=%s&loc=%s&parseTime=true", config.User, config.Pass, config.URL, config.Schema, charset, url.QueryEscape(loc.String()))
	for k, v := range config.Ext {
		if k != "" {
			connectURL = connectURL + "&" + k + "=" + url.QueryEscape(v)
		}
	}

	db, err := sql.Open("mysql", connectURL)
	if err != nil {
		log.Println("Error on initializing database connection,", err.Error())
		return nil, &DBError{"Can't open connection", err}
	}

	c.Infof("db max idle connections:%d,max open connections:%d,charset:%s,ext:%v", config.MaxIdle, config.MaxConn, charset, config.Ext)
	db.SetMaxIdleConns(config.MaxIdle)
	db.SetMaxOpenConns(config.MaxConn)
	db.SetConnMaxLifetime(time.Duration(config.MaxTimeSecond) * time.Second)
	return &Pool{db: db}, nil
}
