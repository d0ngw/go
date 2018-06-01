package orm

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
)

var (
	config = DBConfig{
		User:    "root",
		Pass:    "123456",
		URL:     "127.0.0.1:3306",
		Schema:  "test",
		MaxConn: 100,
		MaxIdle: 10,
	}
	dbpool, err = NewMySQLDBPool(&config)

	setupSQL, _    = ioutil.ReadFile(path.Join("testdata", "setup.sql"))
	teardownSQL, _ = ioutil.ReadFile(path.Join("testdata", "teardown.sql"))
)

type tmodel struct {
	BaseShardEntity
	ID   int64           `column:"id" pk:"Y"`
	Name sql.NullString  `column:"name"`
	Time sql.NullInt64   `column:"create_time"`
	F64  sql.NullFloat64 `column:"f64"`
}

func (tm *tmodel) TableName() string {
	return "tt"
}

type User struct {
	BaseShardEntity
	ID   int64          `column:"id" pk:"Y"`
	Name sql.NullString `column:"name"`
	Age  int64          `column:"age"`
}

func (p *User) TableName() string {
	return "user"
}

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setUp() {
	sqls := strings.Split(string(setupSQL), "--")
	for _, s := range sqls {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		_, err := dbpool.db.Exec(s)
		if err != nil {
			fmt.Println(s)
			panic(err)
		}
	}
}

func teardown() {
	sqls := strings.Split(string(teardownSQL), "--")
	for _, s := range sqls {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		_, err := dbpool.db.Exec(s)
		if err != nil {
			panic(err)
		}
	}
}
