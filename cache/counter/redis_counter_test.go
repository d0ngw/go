package counter

import (
	"fmt"
	"testing"
	"time"

	"os/user"
	"path"

	"github.com/d0ngw/go/cache"
	c "github.com/d0ngw/go/common"
	"github.com/d0ngw/go/orm"
	"github.com/stretchr/testify/assert"
)

type V struct {
	BaseEntity
	Time int64 `column:"ut"`
}

func (p *V) TableName() string {
	return "v"
}

func (p *V) Entity(counterID string, fields Fields) (orm.Entity, error) {
	e, err := p.BaseEntity.ToBaseEntity(counterID, fields)
	if err != nil {
		return nil, err
	}
	return &V{
		BaseEntity: *e,
		Time:       c.UnixMills(time.Now()),
	}, nil
}
func (p *V) ZeroFields() Fields {
	return Fields{"a": int64(1), "b": int64(0)}
}

var r *cache.RedisClient
var dbService orm.ShardDBService

func init() {
	var err error
	config := &orm.DBShardConfig{
		Shards: map[string]*orm.DBConfig{
			"test": {
				User:    "root",
				Pass:    "123456",
				URL:     "127.0.0.1:3306",
				Schema:  "test",
				MaxConn: 100,
				MaxIdle: 10},
		},
		Default: "test",
	}

	shardDBServcie := orm.NewSimpleShardDBService(orm.NewMySQLDBPool)
	shardDBServcie.DBShardConfig = config
	dbService = shardDBServcie

	dbService.Init()

	redisServer := &cache.RedisServer{
		ID:   "test",
		Host: "127.0.0.1",
		Port: 6379,
	}
	var redisConf = cache.RedisConf{
		Servers: []*cache.RedisServer{redisServer},
		Groups:  map[string][]string{"test": {"test"}, "test.sync": {"test"}},
	}

	err = redisConf.Parse()
	if err != nil {
		panic(err)
	}
	r = cache.NewRedisClientWithConf(&redisConf)
	orm.AddMeta(&V{})
}

type persistMock struct {
}

func (p *persistMock) Load(counterID string) (fields Fields, err error) {
	fmt.Println("Load:" + counterID)
	return Fields{"a": int64(1), "b": int64(0)}, nil
}

func (p *persistMock) Del(counterID string) (deleted bool, err error) {
	fmt.Println("Del:" + counterID)
	return true, nil
}

func (p *persistMock) Store(counterID string, fieldAndDelta Fields) error {
	fmt.Printf("Store %s,v:%v:", counterID, fieldAndDelta)
	return nil
}

func TestPersistCounter(t *testing.T) {
	user, err := user.Current()
	assert.Nil(t, err)
	var cacheConf = cache.NewParamConf("test", "c_", 0)
	scripts := &Scripts{
		Update:  path.Join(user.HomeDir, "temp", "lua", "counter_update.lua"),
		SetSync: path.Join(user.HomeDir, "temp", "lua", "counter_update_sync.lua"),
		Evict:   path.Join(user.HomeDir, "temp", "lua", "counter_evict.lua"),
		HgetAll: path.Join(user.HomeDir, "temp", "lua", "counter_getall.lua"),
		Del:     path.Join(user.HomeDir, "temp", "lua", "counter_del.lua"),
	}

	//persist := &persistMock{}
	persist, err := NewDBPersist(func() orm.ShardDBService { return dbService }, &V{})
	assert.Nil(t, err)

	counter := NewPersistRedisCounter("test", func() *cache.RedisClient { return r }, scripts, persist, cacheConf, 10)

	err = counter.scripts.Init()
	assert.Nil(t, err)

	err = counter.Init()
	assert.Nil(t, err)

	testCounter(t, counter)

	//counter.persist, err =
	//assert.Nil(t, err)
	testCounter(t, counter)

	redisCounterSync, err := NewRedisCounterSync(counter, 10, 1, 1, 1)
	assert.Nil(t, err)
	err = redisCounterSync.ScanAll()
	assert.Nil(t, err)

	syncSchedule, err := NewRedisCounterSyncSchedule("test", []*RedisCounterSync{redisCounterSync}, 5)
	assert.Nil(t, err)
	assert.Nil(t, syncSchedule.Init())
	assert.True(t, syncSchedule.Start())
	time.Sleep(time.Duration(5*syncSchedule.scanIntervalSecond) * time.Second)
	assert.True(t, syncSchedule.Stop())
}

func testCounter(t *testing.T, counter *PersistRedisCounter) {
	var err error
	id := "1"
	err = counter.Del(id)
	assert.Nil(t, err)

	fields, err := counter.Get(id)
	assert.Nil(t, err)
	assert.EqualValues(t, 1, fields["a"])
	assert.EqualValues(t, 0, fields["b"])

	err = counter.Incr(id, Fields{"a": 1, "b": 2})
	assert.Nil(t, err)

	fields, err = counter.Get(id)
	assert.Nil(t, err)
	assert.EqualValues(t, 2, fields["a"])
	assert.EqualValues(t, 2, fields["b"])

	err = counter.persist.Store(id, fields)
	assert.Nil(t, err)

	fields, err = counter.Get(id)
	assert.Nil(t, err)
	assert.EqualValues(t, 2, fields["a"])
	assert.EqualValues(t, 2, fields["b"])

	fields, err = counter.Get(id)
	assert.Nil(t, err)
	assert.EqualValues(t, 2, fields["a"])
	assert.EqualValues(t, 2, fields["b"])
}

func TestNoPersistCounter(t *testing.T) {
	var cacheConf = cache.NewParamConf("test", "np_c_", 30)
	counter, err := NewNoPersistRedisCounter("test", r, cacheConf)
	assert.Nil(t, err)

	id := "1"
	var fieldAndDelta = Fields{"a": 1, "b": 2}
	err = counter.Incr(id, fieldAndDelta)
	assert.Nil(t, err)

	reply, err := counter.Get(id)
	assert.Nil(t, err)
	assert.Equal(t, fieldAndDelta, reply)

	err = counter.DelFields(id, "a")
	assert.Nil(t, err)

	reply, err = counter.Get(id)
	assert.Nil(t, err)
	assert.Equal(t, Fields{"b": 2}, reply)

	err = counter.Del(id)
	assert.Nil(t, err)

	reply, err = counter.Get(id)
	assert.Nil(t, err)
	assert.Nil(t, reply)
}
