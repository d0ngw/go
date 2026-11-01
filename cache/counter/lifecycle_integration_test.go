//go:build integration

package counter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

type blockingPersist struct {
	value        atomic.Int64
	storeEntered chan struct{}
	releaseStore chan struct{}
	blocked      bool
}

func (p *blockingPersist) Load(string) (Fields, error) { return Fields{"n": p.value.Load()}, nil }
func (p *blockingPersist) Del(string) (bool, error)    { p.value.Store(0); return true, nil }
func (p *blockingPersist) Store(_ string, fields Fields) error {
	if !p.blocked {
		p.blocked = true
		close(p.storeEntered)
		<-p.releaseStore
	}
	p.value.Store(fields["n"])
	return nil
}

func TestDeleteDuringSyncRejectsStaleAcknowledgement(t *testing.T) {
	persist := &blockingPersist{storeEntered: make(chan struct{}), releaseStore: make(chan struct{})}
	counter := newSyncTestCounter(t, persist)
	require.NoError(t, counter.Incr("1", Fields{"n": 1}))
	param := counter.cacheParam.NewParamKeyWithoutPrefix(counter.counterKey("1"))
	_, err := r.Do(param, func(conn redis.Conn) (interface{}, error) {
		return conn.Do("ZADD", counter.syncSetKey(param.Key()), 1, param.Key())
	})
	require.NoError(t, err)
	scan, err := NewRedisCounterSync(counter, 100, 1, 1, 1)
	require.NoError(t, err)
	var once sync.Once
	release := func() { once.Do(func() { close(persist.releaseStore) }) }
	t.Cleanup(release)
	scanDone := make(chan error, 1)
	go func() { scanDone <- scan.ScanAll() }()
	select {
	case <-persist.storeEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("Store did not start")
	}
	deleteDone := make(chan error, 1)
	go func() { deleteDone <- counter.Del("1") }()
	select {
	case err := <-deleteDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("delete blocked behind persistence")
	}
	require.NoError(t, counter.Incr("1", Fields{"n": 10}))
	release()
	select {
	case err := <-scanDone:
		require.ErrorContains(t, err, "counter generation changed during persistence")
	case <-time.After(2 * time.Second):
		t.Fatal("scan did not finish")
	}
	// The plain backend accepted the obsolete save; Redis must leave the new
	// generation pending so that a later scan can save its current value.
	require.EqualValues(t, 1, persist.value.Load())
	_, err = r.Do(param, func(conn redis.Conn) (interface{}, error) {
		fields, err := redis.StringMap(conn.Do("HGETALL", param.Key()))
		if err != nil {
			return nil, err
		}
		require.Equal(t, "1", fields[counterWriteVer])
		require.Equal(t, "0", fields[counterSyncVer])
		return nil, nil
	})
	require.NoError(t, err)
	fields, err := counter.Get("1")
	require.NoError(t, err)
	require.EqualValues(t, 10, fields["n"])
	require.NoError(t, scan.ScanAll())
	require.EqualValues(t, 10, persist.value.Load())
}

func TestIncrementDuringSyncRemainsPending(t *testing.T) {
	persist := &blockingPersist{storeEntered: make(chan struct{}), releaseStore: make(chan struct{})}
	counter := newSyncTestCounter(t, persist)
	require.NoError(t, counter.Incr("1", Fields{"n": 1}))
	key := counter.counterKey("1")
	param := counter.cacheParam.NewParamKeyWithoutPrefix(key)
	_, err := r.Do(param, func(conn redis.Conn) (interface{}, error) {
		return conn.Do("ZADD", counter.syncSetKey(key), 1, key)
	})
	require.NoError(t, err)
	scan, err := NewRedisCounterSync(counter, 100, 1, 1, 1)
	require.NoError(t, err)
	var once sync.Once
	release := func() { once.Do(func() { close(persist.releaseStore) }) }
	t.Cleanup(release)
	scanDone := make(chan error, 1)
	go func() { scanDone <- scan.ScanAll() }()
	select {
	case <-persist.storeEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("Store did not start")
	}
	// Cache-hit increments stay available while persistence is blocked.
	incrementDone := make(chan error, 1)
	go func() { incrementDone <- counter.Incr("1", Fields{"n": 10}) }()
	select {
	case err := <-incrementDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("increment blocked behind persistence")
	}
	release()
	select {
	case err := <-scanDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("scan did not finish")
	}
	require.EqualValues(t, 1, persist.value.Load())
	_, err = r.Do(param, func(conn redis.Conn) (interface{}, error) {
		fields, err := redis.StringMap(conn.Do("HGETALL", key))
		if err != nil {
			return nil, err
		}
		require.Equal(t, "2", fields[counterWriteVer])
		require.Equal(t, "1", fields[counterSyncVer])
		require.Equal(t, "11", fields["n"])
		return nil, nil
	})
	require.NoError(t, err)
	require.NoError(t, scan.ScanAll())
	require.EqualValues(t, 11, persist.value.Load())
}
