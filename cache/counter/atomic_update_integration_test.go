//go:build integration

package counter

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"testing"

	"github.com/d0ngw/go/cache"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestFailedCounterUpdatesLeaveAllStateUnchanged(t *testing.T) {
	for _, tt := range []struct {
		name, version, a, b string
		changes             []interface{}
	}{
		{"field overflow", "1", "0", strconv.FormatInt(math.MaxInt64, 10), []interface{}{"a", 1, "b", 1}},
		{"field underflow", "1", "0", strconv.FormatInt(math.MinInt64, 10), []interface{}{"a", 1, "b", -1}},
		{"version overflow", strconv.FormatInt(math.MaxInt64, 10), "0", "0", []interface{}{"a", 1}},
		{"malformed field", "1", "0", "invalid", []interface{}{"a", 1, "b", 1}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			param := cache.NewParamConf("test", testPrefix+t.Name()+"_", 0).NewParamKey("h")
			conn, err := r.GetConn(param)
			require.NoError(t, err)
			defer conn.Close()
			syncKey := param.Key() + "_sync"
			_, err = conn.Do("HMSET", param.Key(), "_w", tt.version, "_s", tt.version, "_st", 1, "_g", "original", "a", tt.a, "b", tt.b)
			require.NoError(t, err)
			_, err = conn.Do("ZADD", syncKey, 123, param.Key())
			require.NoError(t, err)
			before, err := redis.StringMap(conn.Do("HGETALL", param.Key()))
			require.NoError(t, err)
			args := []interface{}{param.Key(), syncKey, 456, 0, "__g", "replacement"}
			args = append(args, tt.changes...)
			_, err = DefaultScripts().update.Do(conn, args...)
			require.Error(t, err)
			after, err := redis.StringMap(conn.Do("HGETALL", param.Key()))
			require.NoError(t, err)
			require.Equal(t, before, after)
			score, err := redis.Int64(conn.Do("ZSCORE", syncKey, param.Key()))
			require.NoError(t, err)
			require.EqualValues(t, 123, score)
		})
	}
}

func TestCounterInitialLoadOverflowCreatesNoKeys(t *testing.T) {
	param := cache.NewParamConf("test", testPrefix+t.Name()+"_", 0).NewParamKey("h")
	conn, err := r.GetConn(param)
	require.NoError(t, err)
	defer conn.Close()
	syncKey := param.Key() + "_sync"
	_, err = DefaultScripts().update.Do(conn, param.Key(), syncKey, 1, 1, "__g", "initial", "_a", math.MaxInt64, "a", 1)
	require.Error(t, err)
	for _, key := range []string{param.Key(), syncKey} {
		exists, err := redis.Bool(conn.Do("EXISTS", key))
		require.NoError(t, err)
		require.False(t, exists)
	}
}

func TestCounterArithmeticMatchesSignedInt64(t *testing.T) {
	param := cache.NewParamConf("test", testPrefix+t.Name()+"_", 0).NewParamKey("h")
	conn, err := r.GetConn(param)
	require.NoError(t, err)
	defer conn.Close()
	scripts := DefaultScripts()
	rng := rand.New(rand.NewSource(42))
	cases := [][2]int64{{math.MaxInt64, 0}, {math.MinInt64, 0}, {math.MaxInt64, 1}, {math.MinInt64, -1}, {math.MinInt64, math.MaxInt64}, {math.MaxInt64, math.MinInt64}, {-1, 1}, {0, math.MinInt64}, {9007199254740993, 1}, {-9007199254740993, -1}}
	for i := 0; i < 200; i++ {
		cases = append(cases, [2]int64{int64(rng.Uint64()), int64(rng.Uint64())})
	}
	for _, pair := range cases {
		_, err := conn.Do("HMSET", param.Key(), "_w", 1, "_s", 1, "_st", 1, "_g", "arithmetic", "a", pair[0])
		require.NoError(t, err)
		expected := new(big.Int).Add(big.NewInt(pair[0]), big.NewInt(pair[1]))
		_, updateErr := scripts.update.Do(conn, param.Key(), param.Key()+"_sync", 1, 0, "__g", "unused", "a", pair[1])
		if !expected.IsInt64() {
			require.Error(t, updateErr, fmt.Sprint(pair))
			continue
		}
		require.NoError(t, updateErr, fmt.Sprint(pair))
		actual, err := redis.String(conn.Do("HGET", param.Key(), "a"))
		require.NoError(t, err)
		require.Equal(t, expected.String(), actual, fmt.Sprint(pair))
	}
}

func TestOldGenerationCannotAcknowledgeOrEvictRecreatedCounter(t *testing.T) {
	counter := newSyncTestCounter(t, &syncTestPersist{store: func(string, Fields) error { return nil }})
	require.NoError(t, counter.Incr("1", Fields{"n": 1}))
	key := counter.counterKey("1")
	param := counter.cacheParam.NewParamKeyWithoutPrefix(key)
	conn, err := r.GetConn(param)
	require.NoError(t, err)
	defer conn.Close()
	oldGeneration, err := redis.String(conn.Do("HGET", key, counterGeneration))
	require.NoError(t, err)
	require.NoError(t, counter.Del("1"))
	require.NoError(t, counter.Incr("1", Fields{"n": 10}))
	generation, err := redis.String(conn.Do("HGET", key, counterGeneration))
	require.NoError(t, err)
	require.NotEqual(t, oldGeneration, generation)
	ack, err := redis.Int(counter.scripts.setSync.Do(conn, key, 1, 1, oldGeneration))
	require.NoError(t, err)
	require.Zero(t, ack)
	result, err := redis.Ints(counter.scripts.evict.Do(conn, key, counter.syncSetKey(key), 1, oldGeneration))
	require.NoError(t, err)
	require.Equal(t, []int{0}, result)
	fields, err := counter.Get("1")
	require.NoError(t, err)
	require.EqualValues(t, 10, fields["n"])
	pending, err := redis.Int(conn.Do("ZCARD", counter.syncSetKey(key)))
	require.NoError(t, err)
	require.Equal(t, 1, pending)
}
