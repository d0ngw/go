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

func (p *V) Entity(counterID string, fields Fields) (orm.EntityInterface, error) {
	e, err := p.BaseEntity.BaseEntity(counterID, fields)
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
var dbpool *orm.DBPool

func init() {
	var err error
	config := orm.MysqlDBConfig{
		"root",
		"123456",
		"127.0.0.1:3306",
		"test",
		100,
		10}
	dbpool, err = config.NewDBPool()

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
	err = orm.AddModel(&V{})
	if err != nil {
		panic(err)
	}
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
	counter := NewPersistRedisCounter("test", cacheConf, 10)
	counter.RedisClient = r
	counter.Scripts = &Scripts{
		Update:  path.Join(user.HomeDir, "temp", "lua", "counter_update.lua"),
		SetSync: path.Join(user.HomeDir, "temp", "lua", "counter_update_sync.lua"),
		Evict:   path.Join(user.HomeDir, "temp", "lua", "counter_evict.lua"),
		HgetAll: path.Join(user.HomeDir, "temp", "lua", "counter_getall.lua"),
		Del:     path.Join(user.HomeDir, "temp", "lua", "counter_del.lua"),
	}

	counter.Persist = &persistMock{}

	err = counter.Scripts.Init()
	assert.Nil(t, err)

	err = counter.Init()
	assert.Nil(t, err)

	testCounter(t, counter)

	counter.Persist, err = NewDBPersist(dbpool, &V{})
	assert.Nil(t, err)
	testCounter(t, counter)
}

func testCounter(t *testing.T, counter *PersistRedisCounter) {
	var err error
	id := "1"
	err = counter.Del(id)
	assert.Nil(t, err)

	fields, err := counter.Get(id)
	assert.Nil(t, err)
	assert.Equal(t, 1, fields["a"])
	assert.Equal(t, 0, fields["b"])

	err = counter.Incr(id, Fields{"a": 1, "b": 2})
	assert.Nil(t, err)

	fields, err = counter.Get(id)
	assert.Nil(t, err)
	assert.Equal(t, 2, fields["a"])
	assert.Equal(t, 2, fields["b"])

	err = counter.Persist.Store(id, fields)
	assert.Nil(t, err)
}
