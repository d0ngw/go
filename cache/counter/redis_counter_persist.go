package counter

import (
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
	_, err = orm.AddOrUpdate(oper, entity)
	return
}

// RedisCounterSync sync redis counter to db
type RedisCounterSync struct {
	Name                  string
	redisServers          []*cache.RedisServer
	persistRedisCounter   *PersistRedisCounter
	dbPersist             Persist
	syncSetCacheParam     *cache.ParamConf
	slotMaxItems          int64
	minSyncVersionChanges int64
	minSyncIntervalSecond int64
	evictIntervalSecond   int64
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

	servers, err := persistRedisCounter.redisClient().GetGroupServers(persistRedisCounter.cacheParam.Group())
	if err != nil {
		return nil, err
	}
	return &RedisCounterSync{
		Name:                  persistRedisCounter.Name + ".sync",
		redisServers:          servers,
		persistRedisCounter:   persistRedisCounter,
		dbPersist:             persistRedisCounter.persist,
		syncSetCacheParam:     persistRedisCounter.cacheParam,
		slotMaxItems:          slotMaxItems,
		minSyncVersionChanges: minSyncVersionChanges,
		minSyncIntervalSecond: minSyncIntervalSecond,
		evictIntervalSecond:   evictIntervalSecond,
	}, nil

}

// ScanAll scan all redis server counter sync set
func (p *RedisCounterSync) ScanAll() error {
	for _, redisServer := range p.redisServers {
		c.Infof("begin scan redis %s:%d", redisServer.Host, redisServer.Port)
		for i := 0; i < p.persistRedisCounter.slotsCount; i++ {
			if err := p.scan(redisServer, i); err != nil {
				c.Errorf("scan redis %s:%d slot:%d fail,err:%s", redisServer.Host, redisServer.Port, i, err)
			}
		}
		c.Infof("finish scan redis %s:%d", redisServer.Host, redisServer.Port)
	}
	return nil
}

// Stop stop the scan task
func (p *RedisCounterSync) Stop() {
	atomic.StoreInt32(&p.stop, 1)
}

const (
	counterWriteVer      = "_w"
	counterSyncVer       = "_s"
	counterSyncTimestamp = "_st"
)

func (p *RedisCounterSync) scan(server *cache.RedisServer, slotIndex int) error {
	if atomic.LoadInt32(&p.stop) == 1 {
		c.Infof("stop,skip")
	}
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

	c.Infof("begin scan slotKey:%s,slotIndex length:%d,needEvictItemCount:%d", syncSetSlotKey, originLength, needEvictItemCount)
	if originLength <= 0 {
		return nil
	}

	var removeCounterKey = func(counterKey string, batchRemoved *int64) {
		if removed, err := redis.Int64(conn.Do(cache.ZREM, syncSetSlotKey, counterKey)); err == nil {
			if removed > 0 {
				(*batchRemoved)++
			}
		} else {
			c.Errorf("remove counter key %s in sync set %s fail,err:%s", counterKey, syncSetSlotKey, err)
		}
	}

	st := c.UnixMills(time.Now())
	var syncedCount, evictedCount, processedCount, batchRemoved, nextStart int64
	var batch int64 = 30
	for nextStart = 0; ; {
		if atomic.LoadInt32(&p.stop) == 1 {
			c.Infof("stop,skip")
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
				continue
			}

			if atomic.LoadInt32(&p.stop) == 1 {
				c.Infof("stop,skip")
			}

			fields, err := redis.StringMap(conn.Do(cache.HGETALL, counterKey))
			if err != nil {
				c.Errorf("invalid counter %s", counterKey)
				continue
			}
			if len(fields) == 0 {
				c.Warnf("not exist counter key:%s,remove it from sync set:%s", counterKey, syncSetSlotKey)
				removeCounterKey(counterKey, &batchRemoved)
				continue
			}

			lastWrite := fields[counterWriteVer]
			lastSync := fields[counterSyncVer]
			syncTime := fields[counterSyncTimestamp]
			if lastWrite == "" || lastSync == "" || syncTime == "" {
				c.Warnf("invalid counter key:%s,_w:%s,_s:%s,_st:%s,remove it from sync set:%s", counterKey, lastWrite, lastSync, syncTime, syncSetSlotKey)
				removeCounterKey(counterKey, &batchRemoved)
				continue
			}

			writeVersion, _ := strconv.ParseInt(lastWrite, 10, 64)
			syncVersion, _ := strconv.ParseInt(lastSync, 10, 64)
			syncEcpoch, _ := strconv.ParseInt(syncTime, 10, 64)
			now := c.UnixMills(time.Now())

			var needSync, needEvict bool

			if writeVersion > 0 {
				if syncVersion < writeVersion {
					lastSyncInterval := now - syncEcpoch
					if lastSyncInterval >= p.minSyncIntervalSecond || writeVersion-syncVersion >= p.minSyncVersionChanges {
						needSync = true
					}
				}
			}
			if accessTime > 0 {
				if now-accessTime >= p.evictIntervalSecond {
					needEvict = true
				}
			}
			if needEvictItemCount > 0 {
				needEvict = true
			}
			if needEvict {
				needSync = true
			}

			var synced = false
			if needSync {
				counterID, err := p.persistRedisCounter.parseCounterID(counterKey)
				if err != nil {
					return err
				}
				counterFields, err := p.persistRedisCounter.buildCounterFields(fields)
				if err != nil {
					return err
				}
				err = p.dbPersist.Store(counterID, counterFields)
				synced = err == nil
				if err != nil {
					c.Errorf("sync counter key:%s,success:%v,last sync:%d fail,err:%v", counterKey, synced, writeVersion, err)
				}
				if synced {
					_, err = p.persistRedisCounter.scripts.setSync.Do(conn, counterKey, writeVersion, now)
					if err != nil {
						return err
					}
					syncedCount++
				}
				if err != nil {

				}
				c.Debugf("sync counter key:%s,success:%d,last sync:%d", counterKey, synced, writeVersion)
			}

			if needEvict {
				if !needSync || synced {
					evictResult, err := redis.Ints(p.persistRedisCounter.scripts.evict.Do(conn, counterKey, syncSetSlotKey, writeVersion))
					if err != nil {
						return err
					}
					evicted := evictResult[0]
					c.Debugf("try evict counter key:%s,evicted:%d", counterKey, evicted)
					if evicted == LUATRUE {
						needEvictItemCount--
						evictedCount++
						batchRemoved++
					}
				}
			}
		}
	}
	c.Infof("finish scan slotKey:%s in %d ms,slot length:%d,evicted:%d,synced:%d", syncSetSlotKey, c.UnixMills(time.Now())-st, originLength, evictedCount, syncedCount)
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
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		c.Infof("Start sync task")
		for {
			if atomic.LoadInt32(&p.stop) == 0 {
				for _, syncScan := range p.redisCounterSyncs {
					c.Infof("begin scan %s", syncScan.Name)
					if err := syncScan.ScanAll(); err != nil {
						c.Errorf("scan %s fail,err:%s", syncScan.Name, err)
					}
					c.Infof("finish scan %s", syncScan.Name)
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
	atomic.StoreInt32(&p.stop, 1)
	for _, syncScan := range p.redisCounterSyncs {
		syncScan.Stop()
	}
	close(p.stopChan)
	c.Infof("wait scan task finish...")
	p.wg.Wait()
	c.Infof("all scan task finished")
	return true
}
