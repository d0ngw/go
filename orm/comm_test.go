package orm

import (
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
