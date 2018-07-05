package orm

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
	defaultMetaReg.clean()

	_, err = defaultMetaReg.regModel(&tm)
	checkError(err, true, t, "pointer tm")
}

func TestAdd(t *testing.T) {
	tm := tmodel{Name: sql.NullString{String: "d0ngw", Valid: true}, Time: sql.NullInt64{Int64: time.Now().Unix(), Valid: true}}
	defaultMetaReg.clean()
	_, err = defaultMetaReg.regModel(&tm)
	dboper := &Op{pool: dbpool}

	err = Add(dboper, &tm)
	checkError(err, true, t, "Add")
	if tm.ID <= 0 {
		t.Error("No id")
	}

	tm = tmodel{Name: sql.NullString{String: "d0ngw2", Valid: true}, Time: sql.NullInt64{Int64: time.Now().Unix(), Valid: true}}

	defer func() {
		err := dboper.Rollback()
		t.Logf("rollback error:%v", err)
	}()

	dboper.BeginTx()

	t.Logf("tx:%v", dboper.tx)

	err = Add(dboper, &tm)
	checkError(err, true, t, "Add")
	if tm.ID <= 0 {
		t.Error("No id")
	}
	t.Logf("Add id:%d", tm.ID)

	r, err := Del(dboper, &tm, tm.ID)
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
	tm := tmodel{Name: sql.NullString{String: "d0ngw", Valid: true}, Time: sql.NullInt64{Int64: time.Now().Unix(), Valid: true}}
	tm2 := tmodel{Name: sql.NullString{String: "d0ngw", Valid: true}, Time: sql.NullInt64{Int64: time.Now().Unix(), Valid: true}}

	defaultMetaReg.clean()
	_, err = defaultMetaReg.regModel(&tm)

	dboper := &Op{pool: dbpool}

	err = Add(dboper, &tm)
	checkError(err, true, t, "Add")

	err = Add(dboper, &tm2)
	checkError(err, true, t, "Add")

	defer dboper.Rollback()
	_, err := dboper.DoInTrans(func(tx *sql.Tx) (interface{}, error) {
		tm.Name = sql.NullString{String: "d0ngw1", Valid: true}
		rt, err := Update(dboper, &tm)
		checkError(err, true, t, "Update")
		if !rt {
			t.Error("Update fail", err, tm.ID, rt)
		}
		rt, err = Update(dboper, &tm2)
		checkError(err, true, t, "Update"+fmt.Sprintf("%d", tm2.ID))
		if rt {
			t.Error("No change,but Updated ", err, tm2.ID, rt)
		}
		e, err := Get(dboper, &tm, tm.ID)
		checkError(err, true, t, "Get")
		t.Logf("Get:%v", e)
		return nil, err
	})
	checkError(err, true, t, "Update")

	oldName2 := tm.Name2.String
	tm.Name = sql.NullString{Valid: true, String: "newname"}
	tm.Name2 = sql.NullString{Valid: true, String: "notchange"}

	updated, err := UpdateExcludeColumns(dboper, &tm, "name2")
	assert.NoError(t, err)
	assert.True(t, updated)

	reget, err := Get(dboper, &tm, tm.ID)
	assert.NoError(t, err)
	tm3 := reget.(*tmodel)
	assert.Equal(t, "newname", tm3.Name.String)
	assert.Equal(t, oldName2, tm3.Name2.String)

	rt, err := Del(dboper, &tm, tm.ID)
	checkError(err, true, t, "Del")
	if !rt {
		t.Error("Update fail", err, tm.ID, rt)
	}

	rt, err = Del(dboper, &tm2, tm2.ID)
	checkError(err, true, t, "Del")
	if !rt {
		t.Error("Update fail", err, tm2.ID, rt)
	}
}

func TestUpdateColumns(t *testing.T) {
	defaultMetaReg.clean()
	tm := tmodel{}
	_, err = defaultMetaReg.regModel(&tm)

	tm = tmodel{Name: sql.NullString{String: "d0ngw", Valid: true}, Time: sql.NullInt64{Int64: time.Now().Unix(), Valid: true}}
	dboper := &Op{pool: dbpool}
	err = Add(dboper, &tm)
	checkError(err, true, t, "Add")

	l, err := UpdateColumns(dboper, &tm, " f64 = ?", "", 0.2)
	checkError(err, true, t, "Update")
	t.Logf("update l:%v", l)

	rt, err := Del(dboper, &tm, tm.ID)
	checkError(err, true, t, "Del")
	t.Logf("del rt:%v", rt)
}

func TestGet(t *testing.T) {
	defaultMetaReg.clean()
	tm := tmodel{}
	_, err = defaultMetaReg.regModel(&tm)

	tm = tmodel{Name: sql.NullString{String: "d0ngw", Valid: true}, Time: sql.NullInt64{Int64: time.Now().Unix(), Valid: true}}
	dboper := &Op{pool: dbpool}
	err = Add(dboper, &tm)
	checkError(err, true, t, "Add")

	e, err := Get(dboper, &tm, tm.ID)
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

	cs, err := QueryColumns(dboper, &tm, []string{"id", "create_time", "f64"}, "")
	checkError(err, true, t, "QueryColumns")
	for _, el := range cs {
		t.Logf("el:%#v", el)
	}

	es, err = Query(dboper, &tm, " WHERE ID < 100")
	checkError(err, true, t, "Query")
	for _, el := range es {
		t.Logf("el:%T,%T", el, tm)
	}

	rt, err := Del(dboper, &tm, tm.ID)
	checkError(err, true, t, "Del")
	t.Logf("del rt:%v", rt)
}

func TestCount(t *testing.T) {
	defaultMetaReg.clean()
	tm := tmodel{}
	_, err = defaultMetaReg.regModel(&tm)

	dboper := &Op{pool: dbpool}
	total, err := QueryCount(dboper, &tm, "id", "")
	checkError(err, true, t, "Count")
	t.Logf("countl:%v", total)
}

func TestAddMeta(t *testing.T) {
	defaultMetaReg.clean()
	tm := tmodel{}
	_meta := AddMeta(&tm)
	assert.Equal(t, "github.com/d0ngw/go/orm.tmodel", _meta.Name())
}

func TestShardEntity(t *testing.T) {
	tm := &tmodel{}
	tm.Name = sql.NullString{String: "abc", Valid: true}

	var entity Entity = tm

	shardEntity, ok := entity.(ShardEntity)
	assert.True(t, ok)
	assert.Equal(t, entity, shardEntity)
	assert.True(t, entity == shardEntity)
}
