package counter

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/d0ngw/go/cache"
	c "github.com/d0ngw/go/common"
	"github.com/garyburd/redigo/redis"
)

// PersistRedisCounter use redis implements Counter which be pesisisted by `Persist``
type PersistRedisCounter struct {
	c.BaseService
	RedisClient *cache.RedisClient `inject:"_"`
	Scripts     *Scripts           `inject:"_"`
	Persist     Persist            `inject:"_"`
	cacheParam  *cache.ParamConf
	slotsCount  int
}

// NewPersistRedisCounter create RedisCounter service,
func NewPersistRedisCounter(name string, cacheParam *cache.ParamConf, slotsCount int) *PersistRedisCounter {
	return &PersistRedisCounter{
		BaseService: c.BaseService{SName: name},
		cacheParam:  cacheParam,
		slotsCount:  slotsCount,
	}
}

// Init implements Servcie.Init
func (p *PersistRedisCounter) Init() error {
	if c.HasNil(p.RedisClient, p.Scripts, p.Persist) {
		return fmt.Errorf("RedisClient,Scripts must be set")
	}
	return nil
}

// Incr implements Counter.Incr
func (p *PersistRedisCounter) Incr(counterID string, fieldAndDelta Fields) error {
	if counterID == "" || fieldAndDelta == nil || len(fieldAndDelta) == 0 {
		return fmt.Errorf("counterID and fieldAndDelta must not be empty")
	}
	counterKey := p.counterKey(counterID)
	syncSetKey := p.syncSetKey(counterKey)
	updateArgs := p.updateArgs(syncSetKey, LUAFALSE, fieldAndDelta)

	param := p.cacheParam.NewParamKey(counterKey)

	exist, updated, err := p.updateReply(p.RedisClient.Eval(param, p.Scripts.update, updateArgs...))
	if err != nil {
		return err
	}
	if updated == LUATRUE {
		return nil
	}

	if exist == LUAFALSE {
		fields, err := p.Persist.Load(counterID)
		if err != nil {
			return err
		}
		updateArgs = p.updateArgs(syncSetKey, LUATRUE, p.buildInitFields(fields), fieldAndDelta)
		exist, updated, err = p.updateReply(p.RedisClient.Eval(param, p.Scripts.update, updateArgs...))
		if err != nil {
			return err
		}
		if updated != LUATRUE {
			return fmt.Errorf("update counterID %s fail,exist:%d,updated:%d", counterID, exist, updated)
		}
	}
	return nil
}

// Get implements Counter.Get
func (p *PersistRedisCounter) Get(counterID string) (fields Fields, err error) {
	lastAccessTime := c.UnixMills(time.Now())
	counterKey := p.counterKey(counterID)
	syncSetKey := p.syncSetKey(counterKey)

	getArgs := []interface{}{syncSetKey, strconv.FormatInt(lastAccessTime, 10)}
	param := p.cacheParam.NewParamKey(counterKey)

	reply, err := redis.Strings(p.RedisClient.Eval(param, p.Scripts.hgetAll, getArgs...))
	if err != nil {
		return nil, err
	}
	fields = Fields{}
	for i := 0; i < len(reply); i += 2 {
		k := reply[i]
		if strings.HasPrefix(k, "_") {
			continue
		}
		v, err := strconv.ParseInt(reply[i+1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse counterID %s fail,err:%s", counterID, err)
		}
		fields[k] = v
	}
	if len(fields) == 0 {
		origin, err := p.Persist.Load(counterID)
		if err != nil {
			return nil, err
		}
		if origin == nil {
			return nil, fmt.Errorf("Load counterID %s nil", counterID)
		}
		_, err = p.RedisClient.Eval(param, p.Scripts.update, p.updateArgs(syncSetKey, LUATRUE, origin)...)
		if err != nil {
			c.Errorf("init counterID %s fail,err:%s", counterID, err)
		}
		fields = origin
	}
	return
}

// Del implements Counter.Del
func (p *PersistRedisCounter) Del(counterID string) (err error) {
	_, err = p.Persist.Del(counterID)
	if err != nil {
		return err
	}
	counterKey := p.counterKey(counterID)
	delArgs := []interface{}{p.syncSetKey(counterKey)}
	_, err = p.RedisClient.Eval(p.cacheParam.NewParamKey(counterKey), p.Scripts.del, delArgs...)
	return
}

func (p *PersistRedisCounter) counterKey(counterID string) string {
	return p.cacheParam.KeyPrefix() + "h:" + counterID
}

func (p *PersistRedisCounter) syncSetKey(counterKey string) string {
	slotIndex := c.Fnv32Hashcode(counterKey) % p.slotsCount
	return p.cacheParam.KeyPrefix() + "z.sync:" + strconv.Itoa(slotIndex)
}

func (p *PersistRedisCounter) buildInitFields(fields Fields) Fields {
	initFields := Fields{}
	for k, v := range fields {
		initFields["_"+k] = v
	}
	return initFields
}

func (p *PersistRedisCounter) updateArgs(syncSetKey string, isInit int, fieldAndDelats ...Fields) []interface{} {
	args := []interface{}{syncSetKey, strconv.FormatInt(c.UnixMills(time.Now()), 10), strconv.Itoa(isInit)}
	for _, fieldAndDelat := range fieldAndDelats {
		for k, v := range fieldAndDelat {
			args = append(args, k, strconv.FormatInt(v, 10))
		}
	}
	return args
}

func (p *PersistRedisCounter) updateReply(redisReply interface{}, redisErr error) (exist int, updated int, err error) {
	reply, err := redis.Ints(redisReply, redisErr)
	if err != nil {
		return LUAFALSE, LUAFALSE, err
	}
	if len(reply) < 2 {
		return LUAFALSE, LUAFALSE, fmt.Errorf("Bad reply length:%d", len(reply))
	}
	exist = reply[0]
	updated = reply[1]
	return
}
