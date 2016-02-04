package orm

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

type tmodel struct {
	Id   int64       `column:"id" pk:"Y"`
	Name NullString  `column:"name"`
	Time NullTime    `column:"create_time"`
	I8   NullInt8    `column:"i8"`
	I16  NullInt16   `column:"i16"`
	I32  NullInt32   `column:"i32"`
	F32  NullFloat32 `column:"f32"`
	F64  NullFloat64 `column:"f64"`
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
	tm := tmodel{Name: NullString{"d0ngw", true}, Time: NullTime{time.Now().Local(), true}}
	_modelReg.clean()
	err = _modelReg.RegModel(&tm)
	dboper := &DBOper{db: dbp.DB}

	err = Add(dboper, &tm)
	checkError(err, true, t, "Add")
	if tm.Id <= 0 {
		t.Error("No id")
	}

	tm = tmodel{Name: NullString{"d0ngw2", true}, Time: NullTime{time.Now().Local(), true}}
	defer dboper.Rollback()
	dboper.BeginTx()
	err = Add(dboper, &tm)
	checkError(err, true, t, "Add")
	if tm.Id <= 0 {
		t.Error("No id")
	}
	r, err := Del(dboper, &tm, tm.Id)
	checkError(err, true, t, "Del")
	if !r {
		t.Error("Del fail")
	}
	l, err := DelByCondition(dboper, &tm, "WHERE name = ?", "d0ngw2")
	checkError(err, true, t, "Del")
	t.Logf("Deleted:%v", l)

	l, err = DelByCondition(dboper, &tm, "WHERE name is null ")
	checkError(err, true, t, "Del")
	t.Logf("Deleted:%v", l)

	l, err = DelByCondition(dboper, &tm, "WHERE name =? ", "")
	checkError(err, true, t, "Del")
	t.Logf("Deleted:%v", l)

	err = dboper.Commit()
	checkError(err, true, t, "Add")

}

func TestUpdate(t *testing.T) {
	tm := tmodel{Id: 1, Name: NullString{"d0ngw", true}, Time: NullTime{time.Now(), true}}
	tm2 := tmodel{Id: 4, Name: NullString{"d0ngw", true}, Time: NullTime{time.Now(), true}}
	dboper := &DBOper{db: dbp.DB}
	defer dboper.Rollback()
	_, err := dboper.DoInTrans(func(tx *sql.Tx) (interface{}, error) {
		rt, err := Update(dboper, &tm)
		checkError(err, true, t, "Update")
		if !rt {
			t.Error("Update fail")
		}
		rt, err = Update(dboper, &tm2)
		checkError(err, true, t, "Update")
		if !rt {
			t.Error("Update fail")
		}
		e, err := Get(dboper, &tm, 1)
		checkError(err, true, t, "Update")
		t.Logf("Get:%v", e)
		return nil, err
	})
	checkError(err, true, t, "Update")
}

func TestUpdateColumns(t *testing.T) {
	tm := tmodel{}
	dboper := &DBOper{db: dbp.DB}
	l, err := UpdateColumns(dboper, &tm, " f64 = ?", "", 0.2)
	checkError(err, true, t, "Update")
	t.Logf("update l:%v", l)
}

func TestGet(t *testing.T) {
	tm := tmodel{}
	dboper := &DBOper{db: dbp.DB}
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
