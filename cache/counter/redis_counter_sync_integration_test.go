//go:build integration

package counter

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/d0ngw/go/cache"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

type syncTestPersist struct {
	store func(string, Fields) error
}

func (*syncTestPersist) Load(string) (Fields, error)            { return Fields{"n": 0}, nil }
func (*syncTestPersist) Del(string) (bool, error)               { return true, nil }
func (p *syncTestPersist) Store(id string, fields Fields) error { return p.store(id, fields) }

func newSyncTestCounter(t *testing.T, persist Persist) *PersistRedisCounter {
	t.Helper()
	param := cache.NewParamConf("test", testPrefix+t.Name()+"_", 0)
	counter := NewPersistRedisCounter(t.Name(), func() *cache.RedisClient { return r }, DefaultScripts(), persist, param, 1)
	require.NoError(t, counter.Init())
	return counter
}

func TestSyncPersistenceFailureIsReturned(t *testing.T) {
	failure := errors.New("test persistence unavailable")
	counter := newSyncTestCounter(t, &syncTestPersist{store: func(string, Fields) error { return failure }})
	require.NoError(t, counter.Incr("1", Fields{"n": 1}))
	scan, err := NewRedisCounterSync(counter, 100, 1, 1, 3600)
	require.NoError(t, err)
	require.ErrorIs(t, scan.ScanAll(), failure)
	param := counter.cacheParam.NewParamKeyWithoutPrefix(counter.counterKey("1"))
	var raw map[string]string
	_, err = r.Do(param, func(conn redis.Conn) (interface{}, error) {
		var err error
		raw, err = redis.StringMap(conn.Do("HGETALL", param.Key()))
		return nil, err
	})
	require.NoError(t, err)
	require.Equal(t, "1", raw["_w"])
	require.Equal(t, "0", raw["_s"], "failed persistence must never be acknowledged")
}

func TestSyncCancellationStopsBeforeNextItem(t *testing.T) {
	var scan *RedisCounterSync
	stores := 0
	counter := newSyncTestCounter(t, &syncTestPersist{store: func(string, Fields) error {
		stores++
		scan.Stop()
		return nil
	}})
	require.NoError(t, counter.Incr("1", Fields{"n": 1}))
	require.NoError(t, counter.Incr("2", Fields{"n": 1}))
	var err error
	scan, err = NewRedisCounterSync(counter, 100, 1, 1, 3600)
	require.NoError(t, err)
	require.ErrorIs(t, scan.ScanAll(), context.Canceled)
	require.Equal(t, 1, stores)
}

func TestEmbeddedCounterScriptsHandleConcurrentInitialization(t *testing.T) {
	counter := newSyncTestCounter(t, &syncTestPersist{store: func(string, Fields) error { return nil }})
	other := NewPersistRedisCounter("other", func() *cache.RedisClient { return r }, DefaultScripts(), counter.persist, counter.cacheParam, 1)
	require.NoError(t, other.Init())
	clients := []*PersistRedisCounter{counter, other}
	const workers = 24
	failures := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		client := clients[i%len(clients)]
		go func() { defer wg.Done(); failures <- client.Incr("shared", Fields{"n": 1}) }()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	fields, err := counter.Get("shared")
	require.NoError(t, err)
	require.EqualValues(t, workers, fields["n"])
}

func TestEmbeddedEvictionRejectsUnpersistedWrites(t *testing.T) {
	counter := newSyncTestCounter(t, &syncTestPersist{store: func(string, Fields) error { return nil }})
	require.NoError(t, counter.Incr("1", Fields{"n": 1}))
	key := counter.counterKey("1")
	param := counter.cacheParam.NewParamKeyWithoutPrefix(key)
	_, err := r.Do(param, func(conn redis.Conn) (interface{}, error) {
		generation, err := redis.String(conn.Do("HGET", key, counterGeneration))
		if err != nil {
			return nil, err
		}
		// A version check alone must not evict a write that has not been persisted.
		reply, err := redis.Ints(counter.scripts.evict.Do(conn, key, counter.syncSetKey(key), 1, generation))
		if err != nil {
			return nil, err
		}
		require.Equal(t, []int{0}, reply)
		if _, err := counter.scripts.setSync.Do(conn, key, 1, 1, generation); err != nil {
			return nil, err
		}
		// Simulate an update occurring after the scanner took its snapshot.
		if _, err := conn.Do("HINCRBY", key, "_w", 1); err != nil {
			return nil, err
		}
		reply, err = redis.Ints(counter.scripts.evict.Do(conn, key, counter.syncSetKey(key), 1, generation))
		require.NoError(t, err)
		require.Equal(t, []int{0}, reply)
		if _, err := counter.scripts.setSync.Do(conn, key, 2, 2, generation); err != nil {
			return nil, err
		}
		reply, err = redis.Ints(counter.scripts.evict.Do(conn, key, counter.syncSetKey(key), 2, generation))
		require.NoError(t, err)
		require.Equal(t, []int{1}, reply)
		return nil, err
	})
	require.NoError(t, err)
}
