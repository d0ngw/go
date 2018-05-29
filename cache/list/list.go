// Package list has some list cache implemented by redis
package list

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/d0ngw/go/cache"
	"github.com/d0ngw/go/cache/counter"
	c "github.com/d0ngw/go/common"
	"github.com/d0ngw/go/orm"
	"github.com/garyburd/redigo/redis"
)

// load const params
const (
	MaxLoadBatch = 100
)

type pairInt64 [2]int64

// Entity define the list entity interface
type Entity interface {
	orm.Entity
	// GetID return the id
	GetID() int64
	// GetOwnerID return the owner id
	GetOwnerID() string
	// GetTargetID return the target id
	GetTargetID() int64
}

// BaseEntity define the base list entity
type BaseEntity struct {
	ID         int64  `column:"id" pk:"Y"` //auto increment id
	OwnerID    string `column:"o_id"`
	TargetID   int64  `column:"t_id"`
	CreateTime int64  `column:"ct"`
}

// TableName implement Entity.TableName()
func (p *BaseEntity) TableName() string {
	panic("please override this method")
}

// GetID implements Entity.GetID()
func (p *BaseEntity) GetID() int64 {
	return p.ID
}

// GetOwnerID implement Entity.GetOwnerID
func (p *BaseEntity) GetOwnerID() string {
	return p.OwnerID
}

// GetTargetID implement Entity.GetTargetID()
func (p *BaseEntity) GetTargetID() int64 {
	return p.TargetID
}

// CounterEntity is the list counter
type CounterEntity struct {
	counter.BaseEntity
}

// Entity implements  EntityCounter.Entity
func (p *CounterEntity) Entity(counterID string, fields counter.Fields) (orm.Entity, error) {
	e, err := p.BaseEntity.ToBaseEntity(counterID, fields)
	if err != nil {
		return nil, err
	}
	return &CounterEntity{
		BaseEntity: *e,
	}, nil
}

// ZeroFields implements the EntityCounter.ZeroFields
func (p *CounterEntity) ZeroFields() counter.Fields {
	return counter.Fields{"v": int64(0)}
}

// Cache define the list cache
type Cache struct {
	entityPrototype     Entity
	dbService           orm.DBService
	redisClient         *cache.RedisClient
	listCacheParam      *cache.ParamConf
	listOwnerCacheParam *cache.ParamConf
	maxListCount        int64
	targetIDAsScore     bool
	counterService      counter.Counter
}

// NewCache create new Cache
func NewCache(entityProtoType Entity, dbService orm.DBService, redisClient *cache.RedisClient, listCacheParam *cache.ParamConf, maxListCount int64, targetIDAsScore bool, counter counter.Counter) (*Cache, error) {
	if c.HasNil(entityProtoType, dbService, redisClient, listCacheParam, counter) || maxListCount <= 0 {
		return nil, errors.New("dbService,redisClient,listCacheParam and counter must not be nil,and maxtListCount must be >0")
	}

	return &Cache{
		entityPrototype:     entityProtoType,
		dbService:           dbService,
		redisClient:         redisClient,
		listCacheParam:      listCacheParam,
		listOwnerCacheParam: listCacheParam.NewWithKeyPrefix(":otid_"),
		maxListCount:        maxListCount,
		targetIDAsScore:     targetIDAsScore,
		counterService:      counter,
	}, nil
}

func (p *Cache) ownerAndTargetKey(ownerID string, targetID int64) string {
	return fmt.Sprintf("%s_%d", ownerID, targetID)
}

// Add entity to list cache
func (p *Cache) Add(entity Entity) (bool, error) {
	if entity == nil || reflect.TypeOf(entity) != reflect.TypeOf(p.entityPrototype) {
		return false, errors.New("invalid entity")
	}
	ownerKey := p.ownerAndTargetKey(entity.GetOwnerID(), entity.GetTargetID())
	lockKey := ":lock_" + ownerKey
	locked, err := cache.TryLock(lockKey, 6, p.listOwnerCacheParam, p.redisClient)
	if err != nil {
		return false, err
	}
	if !locked {
		c.Warnf("can't lock %s,skip add", lockKey)
		return false, nil
	}
	defer cache.UnLock(lockKey, p.listOwnerCacheParam, p.redisClient)
	defer p.redisClient.Del(p.listOwnerCacheParam.NewParamKey(ownerKey))

	dbOper, err := p.dbService.NewOp()
	if err != nil {
		return false, err
	}

	err = orm.Add(dbOper, entity)
	if err != nil || entity.GetID() <= 0 {
		c.Errorf("add list entity %#v fail,err:%s", entity, err)
		return false, err
	}

	// counter +1
	e := p.counterService.Incr(entity.GetOwnerID(), counter.Fields{"v": 1})
	if e != nil {
		c.Errorf("incr counter for %s fail,err:%s", entity.GetOwnerID(), e)
	}
	var scoreID int64
	if p.targetIDAsScore {
		scoreID = entity.GetTargetID()
	} else {
		scoreID = entity.GetID()
	}

	listKey := p.listCacheParam.NewParamKey(entity.GetOwnerID())
	p.addToRedisList(listKey, 1, []*pairInt64{&pairInt64{entity.GetTargetID(), scoreID}})
	return true, nil
}

// Del delete the ownerID, targetID from list cache
func (p *Cache) Del(ownerID string, targetID int64) (bool, error) {
	dbOper, err := p.dbService.NewOp()
	if err != nil {
		return false, err
	}
	defer p.redisClient.Del(p.listOwnerCacheParam.NewParamKey(p.ownerAndTargetKey(ownerID, targetID)))

	deleted, err := orm.DelByCondition(dbOper, p.entityPrototype, " WHERE o_id = ? AND t_id = ?", ownerID, targetID)
	if err != nil {
		return false, err
	}

	if deleted <= 0 {
		return false, nil
	}

	p.counterService.Incr(ownerID, counter.Fields{"v": -deleted})
	listKey := p.listCacheParam.NewParamKey(ownerID)
	deleted, lastID, length, _ := p.delFromRedisList(listKey, targetID)
	if deleted > 0 {
		if length < p.maxListCount && lastID > 0 {
			preLoad, err := p.loadListFromDB(ownerID, 0, p.maxListCount-length, lastID)
			if err != nil {
				c.Errorf("try to load more list for %s fail,err:%s", ownerID, err)
			} else {
				_, err = p.addToRedisList(listKey, 1, preLoad)
				if err != nil {
					c.Errorf("try to add pre load list for %s fail,err:%s", ownerID, err)
				}
			}
		}
	}
	return true, nil
}

// GetIDForOwnerTarget query the Entity.ID with ownerID and targetID
func (p *Cache) GetIDForOwnerTarget(ownerID string, targetID int64) (id int64, ok bool, err error) {
	key := p.listOwnerCacheParam.NewParamKey(p.ownerAndTargetKey(ownerID, targetID))
	id, ok, err = p.redisClient.GetInt64(key)
	if err != nil {
		return
	}
	if !ok {
		id, ok, err = p.getIDByOwnerAndTarget(ownerID, targetID)
		if err != nil {
			return
		}
		if ok {
			p.redisClient.Set(key, id)
		} else {
			p.redisClient.Set(key, -1)
		}
	}
	if id > 0 {
		ok = true
	}
	return
}

// GetCount query the count of the ownerID's list
func (p *Cache) GetCount(ownerID string) (count int64, err error) {
	vals, err := p.counterService.Get(ownerID)
	if err != nil {
		return
	}
	count, _ = vals["v"]
	if count < 0 {
		c.Warnf("invalid list %s count:%d", ownerID, count)
		count = 0
	}
	return
}

// LoadList load targetId from list cache
func (p *Cache) LoadList(ownerID string, page, pageSize, cursor int64) (total int64, ids []int64, err error) {
	total, targetAndScores, err := p.loadListWithScoreID(ownerID, page, pageSize, cursor)
	if err != nil {
		return
	}
	ids = make([]int64, 0, len(targetAndScores))
	for _, v := range targetAndScores {
		ids = append(ids, v[0])
	}
	return
}

func (p *Cache) addToRedisList(listKey cache.Param, keyMustExist int, targetAndScores []*pairInt64) (bool, error) {
	if len(targetAndScores) == 0 {
		return false, nil
	}

	conn, err := p.redisClient.GetConn(listKey)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	keyAndParams := []interface{}{listKey.Key(), keyMustExist, listKey.Expire()}
	for _, v := range targetAndScores {
		keyAndParams = append(keyAndParams, v[1], v[0])
	}

	reply, err := redis.Ints(addScript.Do(conn, keyAndParams...))
	if err != nil {
		return false, err
	}
	updated := reply[1]
	return updated == 1, nil
}

func (p *Cache) delFromRedisList(listKey cache.Param, targetID int64) (deleted, lastTargetID, length int64, err error) {
	conn, err := p.redisClient.GetConn(listKey)
	if err != nil {
		return
	}
	defer conn.Close()
	reply, err := delScript.Do(conn, listKey.Key(), listKey.Expire(), targetID)
	if err != nil {
		c.Errorf("del listKey:%s ,targetID:%d fail,err:%s", listKey.Key(), targetID, err)
		return
	}

	replySlice := (reply).([]interface{})
	deleted, err = redis.Int64(replySlice[0], nil)
	if err != nil {
		return
	}
	last := replySlice[1].([]interface{})
	if len(last) > 0 {
		lastTargetID, err = redis.Int64(last, nil)
		if err != nil {
			return
		}
	}
	length, err = redis.Int64(replySlice[2], nil)
	if err != nil {
		return
	}
	return
}

func (p *Cache) loadListFromDB(ownerID string, page int64, pageSize int64, cursor int64) (targetAndScores []*pairInt64, err error) {
	if cursor > 0 {
		return p.loadByCursor(ownerID, pageSize, cursor)
	} else if page > 0 {
		return p.loadByPage(ownerID, page, pageSize)
	}
	return nil, errors.New("the cursor and page must not both <=0")
}

func (p *Cache) loadByCursor(ownerID string, pageSize int64, cursor int64) (targetAndScores []*pairInt64, err error) {
	if p.targetIDAsScore {
		return p.loadIDs("WHERE o_id= ? AND  t_id < ? ORDER BY t_id DESC LIMIT ?", ownerID, cursor, pageSize)
	}
	return p.loadIDs("WHERE o_id= ? AND  id < ? ORDER BY id DESC LIMIT ?", ownerID, cursor, pageSize)
}

func (p *Cache) loadByPage(ownerID string, page, pageSize int64) (targetAndScores []*pairInt64, err error) {
	from := (page - 1) * pageSize
	if p.targetIDAsScore {
		return p.loadIDs("WHERE o_id = ? ORDER BY t_id DESC LIMIT ?, ?", ownerID, from, pageSize)
	}
	return p.loadIDs("WHERE o_id= ? ORDER BY id DESC LIMIT ?,?", ownerID, from, pageSize)
}

func (p *Cache) loadIDs(contition string, params ...interface{}) (targetAndScores []*pairInt64, err error) {
	dbOper, err := p.dbService.NewOp()
	if err != nil {
		return nil, err
	}

	var vals []orm.Entity

	vals, err = orm.QueryColumns(dbOper, p.entityPrototype, []string{"t_id", "id"}, contition, params...)
	if err != nil {
		return nil, err
	}

	for _, val := range vals {
		v := val.(Entity)
		targetAndScore := pairInt64{}
		if p.targetIDAsScore {
			targetAndScore[0] = v.GetTargetID()
			targetAndScore[1] = v.GetTargetID()
		} else {
			targetAndScore[0] = v.GetTargetID()
			targetAndScore[1] = v.GetID()
		}
		targetAndScores = append(targetAndScores, &targetAndScore)
	}
	return
}

func (p *Cache) getIDByOwnerAndTarget(ownerID string, targetID int64) (id int64, ok bool, err error) {
	dbOper, err := p.dbService.NewOp()
	if err != nil {
		return
	}
	vals, err := orm.QueryColumns(dbOper, p.entityPrototype, []string{"id"}, "WHERE o_id =? AND t_id = ?", ownerID, targetID)
	if err != nil {
		return
	}
	if len(vals) > 0 {
		ok = true
	}
	v := vals[0].(Entity)
	id = v.GetID()
	return
}

func (p *Cache) loadListWithScoreID(ownerID string, page, pageSize, cursor int64) (total int64, targetAndScores []*pairInt64, err error) {
	total, err = p.GetCount(ownerID)
	if err != nil {
		return
	}
	if total <= 0 {
		total = 0
		return
	}

	start := (page - 1) * pageSize
	end := start + pageSize - 1
	if start >= total || start < 0 || end < 0 {
		return
	}

	listKey := p.listCacheParam.NewParamKey(ownerID)
	exist, err := p.redisClient.Exists(listKey)
	if err != nil {
		return
	}
	var loadedIDs []*pairInt64
	if !exist {
		var batch = p.maxListCount
		if batch > 100 {
			batch = 100
		}
		loadedIDs, err = p.loadListFromDB(ownerID, 1, batch, 0)
		if err != nil {
			return
		}
		_, err = p.addToRedisList(listKey, 0, loadedIDs)
		if err != nil {
			return
		}
	}

	var (
		curListCount int64
		reply        interface{}
	)
	reply, err = p.redisClient.Do(listKey, func(conn redis.Conn) (interface{}, error) {
		return conn.Do(cache.ZCARD, listKey.Key())
	})
	if err != nil {
		return
	}
	if reply != nil {
		curListCount, _ = redis.Int64(curListCount, err)
	}

	targetAndScores = make([]*pairInt64, 0, pageSize)
	startInCacheList := curListCount > 0 && start < p.maxListCount && start < curListCount
	if startInCacheList {
		reply, err = p.redisClient.Do(listKey, func(conn redis.Conn) (interface{}, error) {
			conn.Send(cache.ZRANGEWITHSCORES, listKey.Key(), start, end)
			if listKey.Expire() > 0 {
				conn.Send(cache.EXPIRE, listKey.Key(), listKey.Expire())
			}
			conn.Flush()
			reply, err := conn.Receive()
			conn.Receive()
			return reply, err
		})
		if err != nil {
			return
		}
		ids, _ := redis.Strings(reply, err)
		if len(ids) > 0 {
			for i := 0; i < len(ids); i += 2 {
				tid, _ := strconv.ParseInt(ids[i], 10, 64)
				score, _ := strconv.ParseInt(ids[i+1], 10, 64)
				pair := &pairInt64{tid, -score}
				targetAndScores = append(targetAndScores, pair)
			}
		}

		retLen := len(targetAndScores)
		if retLen > 0 {
			if int64(retLen) < pageSize {
				if total > curListCount {
					last := targetAndScores[retLen-1]
					cursor := last[1]
					limit := pageSize - int64(retLen)
					loadedIDs, err = p.loadListFromDB(ownerID, 0, limit, cursor)
					if err != nil {
						return
					}
					targetAndScores = append(targetAndScores, loadedIDs...)
					p.addToRedisList(listKey, 1, loadedIDs)
				}
			}
			return
		}
	}
	targetAndScores, err = p.loadListFromDB(ownerID, page, pageSize, cursor)
	return
}
