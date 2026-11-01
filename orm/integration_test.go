//go:build integration

package orm

import (
	_ "embed"
	"fmt"
	"os"
	"testing"

	"github.com/d0ngw/go/internal/testutil"
)

//go:embed testdata/setup.sql
var integrationSchema string

var config DBConfig
var dbpool *Pool
var err error

func TestMain(m *testing.M) {
	os.Exit(runIntegrationTests(m))
}

func runIntegrationTests(m *testing.M) (code int) {
	database, setupErr := testutil.NewDatabase("orm", integrationSchema)
	if setupErr != nil {
		fmt.Fprintln(os.Stderr, setupErr)
		return 1
	}
	defer testutil.CheckCleanup(&code, database.Close)
	config = DBConfig{User: database.User, Pass: database.Password, URL: database.Addr, Schema: database.Name, MaxConn: 10, MaxIdle: 2}
	dbpool, err = NewMySQLDBPool(&config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer testutil.CheckCleanup(&code, dbpool.Close)
	return m.Run()
}
