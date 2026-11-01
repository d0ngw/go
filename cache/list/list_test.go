//go:build integration

package list

import (
	"errors"
	"fmt"
	"github.com/d0ngw/go/internal/testutil"
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	"github.com/d0ngw/go/cache"
	"github.com/d0ngw/go/cache/counter"
	"github.com/d0ngw/go/orm"
	"github.com/stretchr/testify/assert"
)

var r *cache.RedisClient
var dbService orm.ShardDBService

type testCounterEntity struct {
	CounterEntity
}

func (p *testCounterEntity) TableName() string {
	return "v"
}

func (p *testCounterEntity) Entity(counterID string, fields counter.Fields) (orm.Entity, error) {
	e, err := p.CounterEntity.ToBaseEntity(counterID, fields)
	if err != nil {
		return nil, err
	}
	return &testCounterEntity{
		CounterEntity: *e,
	}, nil
}

type testListEntity struct {
	BaseEntity
}

func (p *testListEntity) TableName() string {
	return "list"
}

func TestMain(m *testing.M) { os.Exit(runIntegrationTests(m)) }

func runIntegrationTests(m *testing.M) (code int) {
	database, err := testutil.NewDatabase("list", testutil.CacheSchema)
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
	orm.AddMeta(&testCounterEntity{})
	orm.AddMeta(&testListEntity{})
	return m.Run()
}

var testPrefix string

func TestList(t *testing.T) {
	listCacheParm := cache.NewParamConf("test", testPrefix+"list_", 30)
	counterCacheParam := cache.NewParamConf("test", testPrefix+"list_c_", 30)
	scripts := counter.DefaultScripts()

	persist, err := counter.NewDBPersist(func() orm.ShardDBService { return dbService }, &testCounterEntity{})
	require.NoError(t, err)
	counter := counter.NewPersistRedisCounter("test", func() *cache.RedisClient { return r }, scripts, persist, counterCacheParam, 10)
	err = counter.Init()
	require.NoError(t, err)

	// id as score
	listCache, err := NewCache(&testListEntity{}, func() orm.ShardDBService { return dbService }, func() *cache.RedisClient { return r }, listCacheParm, 500, false, counter)
	require.NoError(t, err)

	for i := 1; i <= 100; i++ {
		toAdd := &testListEntity{BaseEntity: BaseEntity{OwnerID: "d0ngw", TargetID: int64(i)}}
		succ, err := listCache.Add(toAdd)
		assert.NoError(t, err)
		assert.True(t, succ)
		succ, err = listCache.Add(&testListEntity{BaseEntity: BaseEntity{OwnerID: "d0ngw", TargetID: int64(i)}})
		assert.Error(t, err)
		assert.False(t, succ)
	}

	total, err := listCache.GetCount("d0ngw")
	require.NoError(t, err)
	assert.EqualValues(t, 100, total)

	total, ids, err := listCache.LoadList("d0ngw", 1, 10, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 100, total)
	assert.EqualValues(t, 10, len(ids))
	for i, v := range ids {
		assert.EqualValues(t, 100-i, v)
	}

	total, targetScores, err := listCache.LoadListWithScore("d0ngw", 1, 10, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 100, total)
	assert.EqualValues(t, 10, len(targetScores))
	for i, v := range targetScores {
		assert.EqualValues(t, 100-i, v[0])
		t.Logf("tareget id:%d score id:%d", v[0], v[1])
	}

	for i := 1; i <= 100; i++ {
		succ, err := listCache.Del("d0ngw", int64(i))
		require.NoError(t, err)
		assert.True(t, succ)
	}

	total, err = listCache.GetCount("d0ngw")
	require.NoError(t, err)
	assert.EqualValues(t, 0, total)

	// target id as score
	listCache, err = NewCache(&testListEntity{}, func() orm.ShardDBService { return dbService }, func() *cache.RedisClient { return r }, listCacheParm, 5, true, counter)
	require.NoError(t, err)

	for i := 1; i <= 100; i++ {
		toAdd := &testListEntity{BaseEntity: BaseEntity{OwnerID: "d0ngw-t", TargetID: int64(i)}}
		succ, err := listCache.Add(toAdd)
		assert.NoError(t, err)
		assert.True(t, succ)
		succ, err = listCache.Add(&testListEntity{BaseEntity: BaseEntity{OwnerID: "d0ngw-t", TargetID: int64(i)}})
		assert.Error(t, err)
		assert.False(t, succ)
	}

	total, err = listCache.GetCount("d0ngw-t")
	require.NoError(t, err)
	assert.EqualValues(t, 100, total)

	total, ids, err = listCache.LoadList("d0ngw-t", 1, 10, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 100, total)
	assert.EqualValues(t, 10, len(ids))
	for i, v := range ids {
		assert.EqualValues(t, 100-i, v)
	}

	total, targetScores, err = listCache.LoadListWithScore("d0ngw-t", 1, 100, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 100, total)
	assert.EqualValues(t, 100, len(targetScores))
	for i, v := range targetScores {
		assert.EqualValues(t, 100-i, v[0])
		t.Logf("tareget id:%d score id:%d", v[0], v[1])
	}

	for i := 1; i <= 100; i++ {
		succ, err := listCache.Del("d0ngw-t", int64(i))
		require.NoError(t, err)
		assert.True(t, succ)
	}

	total, err = listCache.GetCount("d0ngw-t")
	require.NoError(t, err)
	assert.EqualValues(t, 0, total)
}
