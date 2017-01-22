package orm

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

type tmodel struct {
	Id   int64           `column:"id" pk:"Y"`
	Name sql.NullString  `column:"name"`
	Time sql.NullInt64   `column:"create_time"`
	F64  sql.NullFloat64 `column:"f64"`
}

func (tm tmodel) TableName() string {
	return "tt"
}

type mf float64

func (tm mf) TableName() string {
	return "t"
}

func checkError(err error, noError bool, t *testing.T, msg string) {
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	if noError && err != nil {
		t.Error(msg, "Have error")
	} else if !noError && err == nil {
		t.Error(msg, "Don't have error")
	}
}

func TestReflect(t *testing.T) {
	var err error
	tm := tmodel{}
	_modelReg.clean()
	err = _modelReg.RegModel(&tm)
	checkError(err, true, t, "pointer tm")
}

func TestAdd(t *testing.T) {
	tm := tmodel{Name: sql.NullString{"d0ngw", true}, Time: sql.NullInt64{time.Now().Unix(), true}}
	_modelReg.clean()
	err = _modelReg.RegModel(&tm)
	dboper := &DBOper{db: dbpool.db}

	err = Add(dboper, &tm)
	checkError(err, true, t, "Add")
	if tm.Id <= 0 {
		t.Error("No id")
	}

	tm = tmodel{Name: sql.NullString{"d0ngw2", true}, Time: sql.NullInt64{time.Now().Unix(), true}}

	defer func() {
		err := dboper.Rollback()
		t.Logf("rollback error:%v", err)
	}()

	dboper.BeginTx()

	t.Logf("tx:%d", dboper.tx)

	err = Add(dboper, &tm)
	checkError(err, true, t, "Add")
	if tm.Id <= 0 {
		t.Error("No id")
	}
	t.Logf("Add id:%d", tm.Id)

	r, err := Del(dboper, &tm, tm.Id)
	checkError(err, true, t, "Del")
	if !r {
		t.Error("Del fail")
	}
	t.Logf("By id Deleted:%v", r)

	l, err := DelByCondition(dboper, &tm, "WHERE name = ?", "d0ngw2")
	checkError(err, true, t, "Del")
	t.Logf("By name Deleted:%v", l)

	l, err = DelByCondition(dboper, &tm, "WHERE name is null ")
	checkError(err, true, t, "Del")
	t.Logf("By name is null Deleted:%v", l)

	l, err = DelByCondition(dboper, &tm, "WHERE name =? ", "d0ngw")
	checkError(err, true, t, "Del")
	t.Logf("By name is '' Deleted:%v", l)

	err = dboper.Commit()
	checkError(err, true, t, "Add")
}

func TestUpdate(t *testing.T) {
	tm := tmodel{Name: sql.NullString{"d0ngw", true}, Time: sql.NullInt64{time.Now().Unix(), true}}
	tm2 := tmodel{Name: sql.NullString{"d0ngw", true}, Time: sql.NullInt64{time.Now().Unix(), true}}

	_modelReg.clean()
	err = _modelReg.RegModel(&tm)

	dboper := &DBOper{db: dbpool.db}

	err = Add(dboper, &tm)
	checkError(err, true, t, "Add")

	err = Add(dboper, &tm2)
	checkError(err, true, t, "Add")

	defer dboper.Rollback()
	_, err := dboper.DoInTrans(func(tx *sql.Tx) (interface{}, error) {
		tm.Name = sql.NullString{"d0ngw1", true}
		rt, err := Update(dboper, &tm)
		checkError(err, true, t, "Update")
		if !rt {
			t.Error("Update fail", err, tm.Id, rt)
		}
		rt, err = Update(dboper, &tm2)
		checkError(err, true, t, "Update"+fmt.Sprint("%d", tm2.Id))
		if rt {
			t.Error("No change,but Updated ", err, tm2.Id, rt)
		}
		e, err := Get(dboper, &tm, tm.Id)
		checkError(err, true, t, "Get")
		t.Logf("Get:%v", e)
		return nil, err
	})
	checkError(err, true, t, "Update")

	rt, err := Del(dboper, &tm, tm.Id)
	checkError(err, true, t, "Del")
	if !rt {
		t.Error("Update fail", err, tm.Id, rt)
	}

	rt, err = Del(dboper, &tm2, tm2.Id)
	checkError(err, true, t, "Del")
	if !rt {
		t.Error("Update fail", err, tm2.Id, rt)
	}
}

func TestUpdateColumns(t *testing.T) {
	tm := tmodel{}
	dboper := &DBOper{db: dbpool.db}
	l, err := UpdateColumns(dboper, &tm, " f64 = ?", "", 0.2)
	checkError(err, true, t, "Update")
	t.Logf("update l:%v", l)
}

func TestGet(t *testing.T) {
	tm := tmodel{}
	dboper := &DBOper{db: dbpool.db}
	e, err := Get(dboper, &tm, 1)
	checkError(err, true, t, "Get")
	t.Logf("e:%v,%T", e, e)

	e, err = Get(dboper, &tm, 5)
	checkError(err, true, t, "Get")
	t.Logf("e:%v", e)

	es, err := Query(dboper, &tm, "")
	checkError(err, true, t, "Query")
	for _, el := range es {
		t.Logf("el:%v", el)
	}

	es, err = Query(dboper, &tm, " WHERE ID < 100")
	checkError(err, true, t, "Query")
	for _, el := range es {
		t.Logf("el:%T,%T", el, tm)
	}
}
