package list

import (
	"os/user"
	"path"
	"testing"

	"github.com/d0ngw/go/cache"
	"github.com/d0ngw/go/cache/counter"
	"github.com/d0ngw/go/orm"
	"github.com/stretchr/testify/assert"
)

var r *cache.RedisClient
var dbService orm.DBService

type testCounterEntity struct {
	CounterEntity
}

func (p *testCounterEntity) TableName() string {
	return "v"
}

type testListEntity struct {
	BaseEntity
}

func (p *testListEntity) TableName() string {
	return "list"
}

func init() {
	var err error
	config := &orm.DBConfig{
		User:    "root",
		Pass:    "123456",
		URL:     "127.0.0.1:3306",
		Schema:  "test",
		MaxConn: 100,
		MaxIdle: 10}

	simpleDBService := orm.NewSimpleDBService(orm.NewMySQLDBPool)
	simpleDBService.Config = config

	dbService = simpleDBService
	dbService.Init()

	redisServer := &cache.RedisServer{
		ID:   "test",
		Host: "127.0.0.1",
		Port: 6379,
	}
	var redisConf = cache.RedisConf{
		Servers: []*cache.RedisServer{redisServer},
		Groups:  map[string][]string{"test": []string{"test"}},
	}

	err = redisConf.Parse()
	if err != nil {
		panic(err)
	}
	r = cache.NewRedisClientWithConf(&redisConf)
	orm.AddMeta(&testCounterEntity{})
	orm.AddMeta(&testListEntity{})
}

func TestList(t *testing.T) {
	listCacheParm := cache.NewParamConf("test", "list_", 30)
	counterCacheParam := cache.NewParamConf("test", "list_c_", 30)
	user, err := user.Current()
	scripts := &counter.Scripts{
		Update:  path.Join(user.HomeDir, "temp", "lua", "counter_update.lua"),
		SetSync: path.Join(user.HomeDir, "temp", "lua", "counter_update_sync.lua"),
		Evict:   path.Join(user.HomeDir, "temp", "lua", "counter_evict.lua"),
		HgetAll: path.Join(user.HomeDir, "temp", "lua", "counter_getall.lua"),
		Del:     path.Join(user.HomeDir, "temp", "lua", "counter_del.lua"),
	}
	err = scripts.Init()
	assert.Nil(t, err)

	persist, err := counter.NewDBPersist(dbService, &testCounterEntity{})
	assert.Nil(t, err)
	counter := counter.NewPersistRedisCounter("test", r, scripts, persist, counterCacheParam, 10)
	err = counter.Init()
	assert.Nil(t, err)
	listCache, err := NewCache(&testListEntity{}, dbService, r, listCacheParm, 500, false, counter)
	assert.Nil(t, err)

	for i := 1; i <= 100; i++ {
		toAdd := &testListEntity{BaseEntity: BaseEntity{OwnerID: "d0ngw", TargetID: int64(i)}}
		succ, err := listCache.Add(toAdd)
		assert.Nil(t, err)
		assert.True(t, succ)
		succ, err = listCache.Add(&testListEntity{BaseEntity: BaseEntity{OwnerID: "d0ngw", TargetID: int64(i)}})
		assert.NotNil(t, err)
		assert.False(t, succ)
	}

	total, err := listCache.GetCount("d0ngw")
	assert.Nil(t, err)
	assert.EqualValues(t, 100, total)

	total, ids, err := listCache.LoadList("d0ngw", 1, 10, 0)
	assert.Nil(t, err)
	assert.EqualValues(t, 100, total)
	assert.EqualValues(t, 10, len(ids))
	for i, v := range ids {
		assert.EqualValues(t, 100-i, v)
	}

	for i := 1; i <= 100; i++ {
		succ, err := listCache.Del("d0ngw", int64(i))
		assert.Nil(t, err)
		assert.True(t, succ)
	}

	total, err = listCache.GetCount("d0ngw")
	assert.Nil(t, err)
	assert.EqualValues(t, 0, total)
}
