package counter

import (
	"context"
	"strings"
	"testing"

	"github.com/d0ngw/go/cache"
	"github.com/stretchr/testify/require"
)

func TestSyncReportsAllShardFailures(t *testing.T) {
	scan := &RedisCounterSync{
		redisServers:        []*cache.RedisServer{{Host: "uninitialized"}},
		persistRedisCounter: &PersistRedisCounter{slotsCount: 2, cacheParam: cache.NewParamConf("test", "unit_", 0)},
	}
	err := scan.ScanAll()
	require.Error(t, err)
	require.Contains(t, err.Error(), "slot 0")
	require.Contains(t, err.Error(), "slot 1")
	require.Equal(t, 2, strings.Count(err.Error(), "no pool"))
}

func TestSyncStopSkipsRedisAccess(t *testing.T) {
	scan := &RedisCounterSync{
		redisServers:        []*cache.RedisServer{nil},
		persistRedisCounter: &PersistRedisCounter{slotsCount: 1},
	}
	scan.Stop()
	require.ErrorIs(t, scan.ScanAll(), context.Canceled)
	require.ErrorIs(t, scan.scan(nil, 0), context.Canceled)
	scan.Stop()
}

func TestSyncScheduleStopIsIdempotent(t *testing.T) {
	schedule, err := NewRedisCounterSyncSchedule("unit", []*RedisCounterSync{{}}, 1)
	require.NoError(t, err)
	require.True(t, schedule.Stop())
	require.True(t, schedule.Stop())
	require.False(t, schedule.Start())
}

// Custom backends only need the original persistence interface.
type plainTestPersist struct{}

func (*plainTestPersist) Load(string) (Fields, error) { return Fields{}, nil }
func (*plainTestPersist) Del(string) (bool, error)    { return true, nil }
func (*plainTestPersist) Store(string, Fields) error  { return nil }

func TestPlainPersistenceIsAccepted(t *testing.T) {
	counter := NewPersistRedisCounter("plain", func() *cache.RedisClient { return nil }, DefaultScripts(), &plainTestPersist{}, cache.NewParamConf("unit", "unit_", 0), 1)
	require.NoError(t, counter.Init())
}
