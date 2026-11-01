package counter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/d0ngw/go/cache"
	c "github.com/d0ngw/go/common"
	"github.com/d0ngw/go/orm"

	"github.com/gomodule/redigo/redis"
)

// EntityCounter entity counter
type EntityCounter interface {
	orm.Entity
	// Fields convert entity to Fields
	Fields() (Fields, error)
	// ZeroFields return zero fields,must not nil
	ZeroFields() Fields
	// Entity convert fields to EntityInterface
	Entity(counterID string, fields Fields) (orm.Entity, error)
}

// BaseEntity is the base counter entity
type BaseEntity struct {
	ID    string `column:"id" pk:"Y" pkAuto:"n"`
	Value string `column:"val"`
}

// TableName implements EntityInterface.TableName,it must be overrided or it will panic
func (p *BaseEntity) TableName() string {
	panic("please overried TableName method")
}

// Fields implements EntityCounter.Fields
func (p *BaseEntity) Fields() (Fields, error) {
	if p.Value == "" {
		return p.ZeroFields(), nil
	}
	fileds := Fields{}
	err := json.Unmarshal([]byte(p.Value), &fileds)
	if err != nil {
		return nil, err
	}
	return fileds, nil
}

// ZeroFields implements EntityCounter.ZeroFields,must be overrided
func (p *BaseEntity) ZeroFields() Fields {
	panic("please override ZeroFields method")
}

// ToBaseEntity convert counterID and fields to BaseEntity
func (p *BaseEntity) ToBaseEntity(counterID string, fields Fields) (*BaseEntity, error) {
	v := &BaseEntity{ID: counterID}
	if fields == nil {
		return v, nil
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	v.Value = string(b)
	return v, nil
}

// DBPersist implements Persist which persist counter to db
type DBPersist struct {
	dbService  func() orm.ShardDBService
	entityType EntityCounter
}

var _ Persist = (*DBPersist)(nil)

// NewDBPersist create DBPersist
func NewDBPersist(dbService func() orm.ShardDBService, entityType EntityCounter) (*DBPersist, error) {
	if c.HasNil(dbService, entityType) {
		return nil, errors.New("dbpool and entityType must not be nil")
	}
	return &DBPersist{
		dbService:  dbService,
		entityType: entityType,
	}, nil
}

// Load  implements Persist.Load
func (p *DBPersist) Load(counterID string) (fields Fields, err error) {
	oper, err := p.dbService().NewOpByEntity(p.entityType, "")
	if err != nil {
		return nil, err
	}
	entity, err := orm.Get(oper, p.entityType, counterID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return p.entityType.ZeroFields(), nil
	}
	if toCounter, ok := entity.(EntityCounter); ok {
		fields, err = toCounter.Fields()
		if err != nil {
			return nil, err
		}
		return
	}
	return nil, fmt.Errorf("%T is not a valid ToCounter", entity)
}

// Del implements Persist.Del
func (p *DBPersist) Del(counterID string) (deleted bool, err error) {
	oper, err := p.dbService().NewOpByEntity(p.entityType, "")
	if err != nil {
		return false, err
	}
	return orm.Del(oper, p.entityType, counterID)
}

// Store implements Persist.Store
func (p *DBPersist) Store(counterID string, fields Fields) (err error) {
	oper, err := p.dbService().NewOpByEntity(p.entityType, "")
	if err != nil {
		return err
	}
	entity, err := p.entityType.Entity(counterID, fields)
	if err != nil {
		return err
	}
	_, _, expect := c.ExtractRefTuple(p.entityType)
	_, _, rel := c.ExtractRefTuple(entity)
	if expect != rel {
		err = fmt.Errorf("expect %v,but is %v", expect, rel)
		return
	}
	_, err = orm.AddOrUpdate(oper, entity)
	return
}

// RedisCounterSync sync redis counter to db
type RedisCounterSync struct {
	Name                  string
	redisServers          []*cache.RedisServer
	persistRedisCounter   *PersistRedisCounter
	slotMaxItems          int64
	minSyncVersionChanges int64
	minSyncIntervalMills  int64
	evictIntervalMills    int64
	stop                  int32
}

// NewRedisCounterSync create new RedisCounterSync
func NewRedisCounterSync(persistRedisCounter *PersistRedisCounter, slotMaxItems, minSyncVersionChanges, minSyncIntervalSecond, evictIntervalSecond int64) (*RedisCounterSync, error) {
	if persistRedisCounter == nil {
		return nil, errors.New("no persistRedisCounter")
	}
	if slotMaxItems <= 0 || minSyncVersionChanges <= 0 || minSyncIntervalSecond <= 0 || evictIntervalSecond <= 0 {
		return nil, errors.New("slotMaxItems and xxxSecond must be >0")
	}

	syncGroup := persistRedisCounter.cacheParam.Group() + ".sync"
	servers, err := persistRedisCounter.redisClient().GetGroupServers(syncGroup)
	if err != nil {
		return nil, err
	}
	return &RedisCounterSync{
		Name:                  persistRedisCounter.Name + ".sync",
		redisServers:          servers,
		persistRedisCounter:   persistRedisCounter,
		slotMaxItems:          slotMaxItems,
		minSyncVersionChanges: minSyncVersionChanges,
		minSyncIntervalMills:  minSyncIntervalSecond * 1000,
		evictIntervalMills:    evictIntervalSecond * 1000,
	}, nil

}

// ScanAll scans every server and returns combined shard failures.
// Stop cancels further work between Redis and persistence operations.
func (p *RedisCounterSync) ScanAll() error {
	var failures []error
	for _, redisServer := range p.redisServers {
		if atomic.LoadInt32(&p.stop) != 0 {
			return errors.Join(append(failures, context.Canceled)...)
		}
		c.Debugf("begin scan redis %s:%d", redisServer.Host, redisServer.Port)
		for i := 0; i < p.persistRedisCounter.slotsCount; i++ {
			if atomic.LoadInt32(&p.stop) != 0 {
				return errors.Join(append(failures, context.Canceled)...)
			}
			if err := p.scan(redisServer, i); err != nil {
				c.Errorf("scan redis %s:%d slot:%d fail,err:%s", redisServer.Host, redisServer.Port, i, err)
				failures = append(failures, fmt.Errorf("scan %s:%d slot %d: %w", redisServer.Host, redisServer.Port, i, err))
				if errors.Is(err, context.Canceled) {
					return errors.Join(failures...)
				}
			}
		}
		c.Debugf("finish scan redis %s:%d", redisServer.Host, redisServer.Port)
	}
	if atomic.LoadInt32(&p.stop) != 0 {
		failures = append(failures, context.Canceled)
	}
	return errors.Join(failures...)
}

// Stop stop the scan task
func (p *RedisCounterSync) Stop() {
	atomic.StoreInt32(&p.stop, 1)
}

const (
	counterGeneration    = "_g"
	counterWriteVer      = "_w"
	counterSyncVer       = "_s"
	counterSyncTimestamp = "_st"
)

func (p *RedisCounterSync) scan(server *cache.RedisServer, slotIndex int) (scanErr error) {
	var failures []error
	defer func() { scanErr = errors.Join(append(failures, scanErr)...) }()
	if atomic.LoadInt32(&p.stop) == 1 {
		return context.Canceled
	}
	counter := p.persistRedisCounter
	syncSetSlotKey := p.persistRedisCounter.syncSetSlotKey(slotIndex)
	conn, err := server.GetConn()
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			c.Errorf("close conn err:%v", err)
		}
	}()

	originLength, err := redis.Int64(conn.Do(cache.ZCARD, syncSetSlotKey))
	if err != nil {
		return err
	}
	overItems := originLength - p.slotMaxItems
	var needEvictItemCount int64
	if overItems > 0 {
		needEvictItemCount = overItems
	}

	if originLength <= 0 {
		return nil
	}

	var removeCounterKey = func(conn redis.Conn, counterKey string, batchRemoved *int64) {
		if removed, err := redis.Int64(conn.Do(cache.ZREM, syncSetSlotKey, counterKey)); err == nil {
			if removed > 0 {
				(*batchRemoved)++
			}
		} else {
			c.Errorf("remove counter key %s in sync set %s fail,err:%s", counterKey, syncSetSlotKey, err)
			failures = append(failures, fmt.Errorf("remove counter %s: %w", counterKey, err))
		}
	}

	st := c.UnixMills(time.Now())
	var syncedCount, evictedCount, processedCount, batchRemoved, nextStart int64
	var batch int64 = 30
	for nextStart = 0; ; {
		if atomic.LoadInt32(&p.stop) == 1 {
			return context.Canceled
		}
		start := nextStart
		if batchRemoved > 0 {
			start = nextStart - batchRemoved
			if start < 0 {
				start = 0
			}
		}
		end := start + batch - 1
		nextStart = end + 1
		length, err := redis.Int64(conn.Do(cache.ZCARD, syncSetSlotKey))
		if err != nil {
			return err
		}
		if start >= length {
			break
		}

		batchRemoved = 0
		counterKeyAndTimeSet, err := redis.Strings(conn.Do(cache.ZRANGE, syncSetSlotKey, start, end, "WITHSCORES"))
		if err != nil {
			return err
		}
		for i := 0; i < len(counterKeyAndTimeSet); i += 2 {
			processedCount++
			counterKey := counterKeyAndTimeSet[i]
			accessTime, err := strconv.ParseInt(counterKeyAndTimeSet[i+1], 10, 64)
			if err != nil {
				c.Errorf("invalid accessTime %s,%d", counterKey, accessTime)
				failures = append(failures, fmt.Errorf("counter %s access time: %w", counterKey, err))
				continue
			}

			if atomic.LoadInt32(&p.stop) == 1 {
				return context.Canceled
			}

			counterID, err := p.persistRedisCounter.parseCounterID(counterKey)
			if err != nil {
				failures = append(failures, err)
				continue
			}
			itemErr := func() error {
				fields, err := redis.StringMap(counterSnapshotScript.Do(conn, counterKey, c.RandomUUID()))
				if err != nil {
					c.Errorf("invalid counter %s", counterKey)
					failures = append(failures, fmt.Errorf("read counter %s: %w", counterKey, err))
					return nil
				}
				if len(fields) == 0 {
					c.Warnf("not exist counter key:%s,remove it from sync set:%s", counterKey, syncSetSlotKey)
					removeCounterKey(conn, counterKey, &batchRemoved)
					return nil
				}

				lastWrite := fields[counterWriteVer]
				lastSync := fields[counterSyncVer]
				syncTime := fields[counterSyncTimestamp]
				if lastWrite == "" || lastSync == "" || syncTime == "" {
					c.Warnf("invalid counter key:%s,_w:%s,_s:%s,_st:%s,remove it from sync set:%s", counterKey, lastWrite, lastSync, syncTime, syncSetSlotKey)
					removeCounterKey(conn, counterKey, &batchRemoved)
					return nil
				}

				writeVersion, _ := strconv.ParseInt(lastWrite, 10, 64)
				syncVersion, _ := strconv.ParseInt(lastSync, 10, 64)
				syncEcpoch, _ := strconv.ParseInt(syncTime, 10, 64)
				now := c.UnixMills(time.Now())

				var needSync, needEvict bool

				if writeVersion > 0 {
					if syncVersion < writeVersion {
						lastSyncInterval := now - syncEcpoch
						if lastSyncInterval >= p.minSyncIntervalMills || writeVersion-syncVersion >= p.minSyncVersionChanges {
							needSync = true
						}
					}
				}
				if accessTime > 0 {
					if now-accessTime >= p.evictIntervalMills {
						needEvict = true
					}
				}
				if needEvictItemCount > 0 {
					needEvict = true
				}
				if needEvict {
					if writeVersion > 0 {
						needSync = true
					}
				}

				var synced = false
				if needSync {
					counterFields, err := p.persistRedisCounter.buildCounterFields(fields)
					if err != nil {
						return err
					}
					c.Infof("sync counter key %s, val %v, _w %d, _s %d, _st %d", counterKey, counterFields, writeVersion, syncVersion, syncEcpoch)
					if atomic.LoadInt32(&p.stop) != 0 {
						return context.Canceled
					}
					err = counter.persist.Store(counterID, counterFields)
					synced = err == nil
					if err != nil {
						c.Errorf("sync counter key:%s,success:%v,last sync:%d fail,err:%v", counterKey, synced, writeVersion, err)
						failures = append(failures, fmt.Errorf("persist counter %s: %w", counterID, err))
					}
					if synced {
						acknowledged, err := redis.Int(p.persistRedisCounter.scripts.setSync.Do(conn, counterKey, writeVersion, now, fields[counterGeneration]))
						if err != nil {
							return err
						}
						if acknowledged != LUATRUE {
							return errors.New("counter generation changed during persistence")
						}
						syncedCount++
					}
				}

				if needEvict {
					if atomic.LoadInt32(&p.stop) != 0 {
						return context.Canceled
					}
					if !needSync || synced {
						evictResult, err := redis.Ints(p.persistRedisCounter.scripts.evict.Do(conn, counterKey, syncSetSlotKey, writeVersion, fields[counterGeneration]))
						if err != nil {
							return err
						}
						if len(evictResult) != 1 {
							return errors.New("invalid counter eviction reply")
						}
						evicted := evictResult[0]
						if evicted == LUATRUE {
							needEvictItemCount--
							evictedCount++
							batchRemoved++
						}
					}
				}
				return nil
			}()
			if itemErr != nil {
				if errors.Is(itemErr, context.Canceled) {
					return itemErr
				}
				failures = append(failures, fmt.Errorf("counter %s: %w", counterID, itemErr))
			}
		}
	}
	if evictedCount > 0 || syncedCount > 0 {
		c.Infof("after scan slotKey:%s in %d ms,evicted:%d,synced:%d", syncSetSlotKey, c.UnixMills(time.Now())-st, evictedCount, syncedCount)
	}
	return nil
}

// RedisCounterSyncSchedule schedul RedisCounterSync task
type RedisCounterSyncSchedule struct {
	c.BaseService
	redisCounterSyncs  []*RedisCounterSync
	scanIntervalSecond int
	stopChan           chan int
	stop               int32
	wg                 sync.WaitGroup
	lock               sync.Mutex
	started            bool
}

// NewRedisCounterSyncSchedule create new RedisCounterSyncSchedule
func NewRedisCounterSyncSchedule(name string, redisCounterSync []*RedisCounterSync, scanIntervalSecond int) (*RedisCounterSyncSchedule, error) {
	if len(redisCounterSync) == 0 || scanIntervalSecond <= 0 {
		return nil, errors.New("invalid params")
	}

	return &RedisCounterSyncSchedule{
		BaseService:        c.BaseService{SName: name},
		redisCounterSyncs:  redisCounterSync,
		scanIntervalSecond: scanIntervalSecond,
		stopChan:           make(chan int, 1),
	}, nil
}

// Init implements Initable.Init
func (p *RedisCounterSyncSchedule) Init() error {
	if len(p.redisCounterSyncs) == 0 || p.scanIntervalSecond <= 0 {
		return errors.New("invalid redisCounterSyncs or scanIntervalSecond")
	}
	return nil
}

// Start implements Servcie.Start()
func (p *RedisCounterSyncSchedule) Start() bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.started || atomic.LoadInt32(&p.stop) != 0 {
		return false
	}
	p.started = true
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		c.Infof("Start sync task")
		for {
			if atomic.LoadInt32(&p.stop) == 0 {
				for _, syncScan := range p.redisCounterSyncs {
					c.Debugf("begin scan %s", syncScan.Name)
					if err := syncScan.ScanAll(); err != nil {
						c.Errorf("scan %s fail,err:%s", syncScan.Name, err)
					}
					c.Debugf("finish scan %s", syncScan.Name)
				}

				timer := time.NewTimer(time.Duration(p.scanIntervalSecond) * time.Second)
				select {
				case <-timer.C:
				case <-p.stopChan:
				}
				timer.Stop()
			} else {
				break
			}
		}
		c.Infof("finish sync task")
	}()
	return true
}

// Stop implements Service.Stop()
func (p *RedisCounterSyncSchedule) Stop() bool {
	p.lock.Lock()
	if atomic.CompareAndSwapInt32(&p.stop, 0, 1) {
		for _, syncScan := range p.redisCounterSyncs {
			syncScan.Stop()
		}
		close(p.stopChan)
	}
	p.lock.Unlock()
	c.Infof("wait scan task finish...")
	p.wg.Wait()
	c.Infof("all scan task finished")
	return true
}
