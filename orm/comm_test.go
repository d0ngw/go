package orm

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path"
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
	ID           int64           `column:"id" pk:"Y"`
	Name         sql.NullString  `column:"name"`
	Time         sql.NullInt64   `column:"create_time"`
	F64          sql.NullFloat64 `column:"f64"`
	tblShardFunc ShardHandler
}

func (tm *tmodel) TableName() string {
	return "tt"
}

func (tm *tmodel) TableShardFunc() ShardHandler {
	return tm.tblShardFunc
}

func (tm *tmodel) SetTableShardFunc(f ShardHandler) {
	tm.tblShardFunc = f
}

func TestMain(m *testing.M) {
	setUp()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setUp() {
	_, err := dbpool.db.Exec(string(setupSQL))
	if err != nil {
		panic(err)
	}
}

func teardown() {
	_, err := dbpool.db.Exec(string(teardownSQL))
	if err != nil {
		panic(err)
	}
}
