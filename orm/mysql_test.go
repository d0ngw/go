package orm

import (
	_ "fmt"
	"testing"
)

func TestMysqlCreateor(t *testing.T) {
	dbp, err := config.NewDBPool()
	if err != nil {
		t.Errorf("Create fail %s", err.Error())
		return
	}
	if dbp == nil {
		t.Error("Create fail", err)
		return
	}

	defer dbp.db.Close()
	err = dbp.db.Ping()
	if err != nil {
		t.Errorf("Ping db fail %s", err)
	}
}
