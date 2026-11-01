//go:build integration

package counter

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/d0ngw/go/internal/testutil"
	"github.com/stretchr/testify/require"
	"os"

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

func TestMain(m *testing.M) { os.Exit(runIntegrationTests(m)) }

func runIntegrationTests(m *testing.M) (code int) {
	database, err := testutil.NewDatabase("counter", testutil.CacheSchema)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer testutil.CheckCleanup(&code, database.Close)
	testPrefix = database.Name + "_"
	config := &orm.DBShardConfig{
		Shards: map[string]*orm.DBConfig{"test": {
			User: database.User, Pass: database.Password, URL: database.Addr,
			Schema: database.Name, MaxConn: 10, MaxIdle: 2,
		}}, Default: "test",
	}
	var pools []*orm.Pool
	service := orm.NewSimpleShardDBService(func(conf *orm.DBConfig) (*orm.Pool, error) {
		pool, err := orm.NewMySQLDBPool(conf)
		if err == nil {
			pools = append(pools, pool)
		}
		return pool, err
	})
	service.DBShardConfig = config
	if err := service.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer testutil.CheckCleanup(&code, func() error {
		var failures []error
		for _, pool := range pools {
			failures = append(failures, pool.Close())
		}
		return errors.Join(failures...)
	})
	dbService = service
	host, port, err := testutil.RedisAddress()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	redisConf := &cache.RedisConf{
		Servers: []*cache.RedisServer{{ID: "test", Host: host, Port: port, Auth: testutil.Env("REDIS_PASSWORD", "")}},
		Groups:  map[string][]string{"test": {"test"}, "test.sync": {"test"}},
	}
	if err := redisConf.Parse(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	r = cache.NewRedisClientWithConf(redisConf)
	defer testutil.CheckCleanup(&code, r.Close)
	defer testutil.CheckCleanup(&code, func() error { return testutil.CleanupRedis(testPrefix) })
	orm.AddMeta(&V{})
	return m.Run()
}

var testPrefix string

func TestPersistCounter(t *testing.T) {
	var cacheConf = cache.NewParamConf("test", testPrefix+"c_", 0)
	scripts := DefaultScripts()

	persist, err := NewDBPersist(func() orm.ShardDBService { return dbService }, &V{})
	require.NoError(t, err)

	counter := NewPersistRedisCounter("test", func() *cache.RedisClient { return r }, scripts, persist, cacheConf, 10)

	err = counter.Init()
	require.NoError(t, err)

	testCounter(t, counter)

	testCounter(t, counter)

	redisCounterSync, err := NewRedisCounterSync(counter, 10, 1, 1, 1)
	require.NoError(t, err)
	err = redisCounterSync.ScanAll()
	require.NoError(t, err)

	syncSchedule, err := NewRedisCounterSyncSchedule("test", []*RedisCounterSync{redisCounterSync}, 5)
	require.NoError(t, err)
	assert.Nil(t, syncSchedule.Init())
	assert.True(t, syncSchedule.Start())
	t.Cleanup(func() { syncSchedule.Stop() })
	assert.True(t, syncSchedule.Stop())
}

func testCounter(t *testing.T, counter *PersistRedisCounter) {
	var err error
	id := "1"
	err = counter.Del(id)
	require.NoError(t, err)

	fields, err := counter.Get(id)
	require.NoError(t, err)
	assert.EqualValues(t, 1, fields["a"])
	assert.EqualValues(t, 0, fields["b"])

	err = counter.Incr(id, Fields{"a": 1, "b": 2})
	require.NoError(t, err)

	fields, err = counter.Get(id)
	require.NoError(t, err)
	assert.EqualValues(t, 2, fields["a"])
	assert.EqualValues(t, 2, fields["b"])

	err = counter.persist.Store(id, fields)
	require.NoError(t, err)

	fields, err = counter.Get(id)
	require.NoError(t, err)
	assert.EqualValues(t, 2, fields["a"])
	assert.EqualValues(t, 2, fields["b"])

	fields, err = counter.Get(id)
	require.NoError(t, err)
	assert.EqualValues(t, 2, fields["a"])
	assert.EqualValues(t, 2, fields["b"])
}

func TestNoPersistCounter(t *testing.T) {
	var cacheConf = cache.NewParamConf("test", testPrefix+"np_c_", 30)
	counter, err := NewNoPersistRedisCounter("test", r, cacheConf)
	require.NoError(t, err)

	id := "1"
	var fieldAndDelta = Fields{"a": 1, "b": 2}
	err = counter.Incr(id, fieldAndDelta)
	require.NoError(t, err)

	reply, err := counter.Get(id)
	require.NoError(t, err)
	assert.Equal(t, fieldAndDelta, reply)

	err = counter.DelFields(id, "a")
	require.NoError(t, err)

	reply, err = counter.Get(id)
	require.NoError(t, err)
	assert.Equal(t, Fields{"b": 2}, reply)

	err = counter.Del(id)
	require.NoError(t, err)

	reply, err = counter.Get(id)
	require.NoError(t, err)
	assert.Nil(t, reply)
}
