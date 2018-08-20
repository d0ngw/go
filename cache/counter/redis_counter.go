package counter

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/d0ngw/go/cache"
	c "github.com/d0ngw/go/common"
	"github.com/gomodule/redigo/redis"
)

// PersistRedisCounter use redis implements Counter which be pesisisted by `Persist``
type PersistRedisCounter struct {
	c.Initable
	Name        string
	redisClient func() *cache.RedisClient
	scripts     *Scripts
	persist     Persist
	cacheParam  *cache.ParamConf
	slotsCount  int
}

// NewPersistRedisCounter create RedisCounter service
func NewPersistRedisCounter(name string, redisClientFunc func() *cache.RedisClient, scripts *Scripts, persist Persist, cacheParam *cache.ParamConf, slotsCount int) *PersistRedisCounter {
	return &PersistRedisCounter{
		Name:        name,
		redisClient: redisClientFunc,
		scripts:     scripts,
		persist:     persist,
		cacheParam:  cacheParam,
		slotsCount:  slotsCount,
	}
}

// Init implements Servcie.Init
func (p *PersistRedisCounter) Init() error {
	if c.HasNil(p.redisClient, p.scripts, p.persist, p.cacheParam) {
		return fmt.Errorf("RedisClient,Scripts must be set")
	}
	if strings.Contains(p.cacheParam.KeyPrefix(), ":") {
		return fmt.Errorf("cacheParam.KeyPrefix %s must does not contain `:`", p.cacheParam.KeyPrefix())
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

	exist, updated, err := p.updateReply(p.redisClient().Eval(param, p.scripts.update, updateArgs...))
	if err != nil {
		return err
	}
	if updated == LUATRUE {
		return nil
	}

	if exist == LUAFALSE {
		fields, err := p.persist.Load(counterID)
		if err != nil {
			return err
		}
		updateArgs = p.updateArgs(syncSetKey, LUATRUE, p.buildInitFields(fields), fieldAndDelta)
		exist, updated, err = p.updateReply(p.redisClient().Eval(param, p.scripts.update, updateArgs...))
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

	reply, err := redis.Strings(p.redisClient().Eval(param, p.scripts.hgetAll, getArgs...))
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
		origin, err := p.persist.Load(counterID)
		if err != nil {
			return nil, err
		}
		if origin == nil {
			return nil, fmt.Errorf("Load counterID %s nil", counterID)
		}
		_, err = p.redisClient().Eval(param, p.scripts.update, p.updateArgs(syncSetKey, LUATRUE, origin)...)
		if err != nil {
			c.Errorf("init counterID %s fail,err:%s", counterID, err)
		}
		fields = origin
	}
	return
}

// Del implements Counter.Del
func (p *PersistRedisCounter) Del(counterID string) (err error) {
	_, err = p.persist.Del(counterID)
	if err != nil {
		return err
	}
	counterKey := p.counterKey(counterID)
	delArgs := []interface{}{p.syncSetKey(counterKey)}
	_, err = p.redisClient().Eval(p.cacheParam.NewParamKey(counterKey), p.scripts.del, delArgs...)
	return
}

func (p *PersistRedisCounter) counterKey(counterID string) string {
	if strings.Contains(counterID, ":") {
		panic(fmt.Errorf("counterID %s must does not contian `:` ", counterID))
	}
	return p.cacheParam.KeyPrefix() + "h:" + counterID
}

func (p *PersistRedisCounter) parseCounterID(counterKey string) (string, error) {
	strs := c.SplitTrimOmitEmpty(counterKey, ":")
	if len(strs) != 2 || strs[1] == "" {
		return "", fmt.Errorf("invalid counter key:%s", counterKey)
	}
	return strs[1], nil
}

func (p *PersistRedisCounter) syncSetKey(counterKey string) string {
	slotIndex := c.Fnv32Hashcode(counterKey) % p.slotsCount
	return p.cacheParam.KeyPrefix() + "z.sync:" + strconv.Itoa(slotIndex)
}

func (p *PersistRedisCounter) syncSetSlotKey(slotIndex int) string {
	return p.cacheParam.KeyPrefix() + "z.sync:" + strconv.Itoa(slotIndex)
}

func (p *PersistRedisCounter) buildInitFields(fields Fields) Fields {
	initFields := Fields{}
	for k, v := range fields {
		initFields["_"+k] = v
	}
	return initFields
}

func (p *PersistRedisCounter) buildCounterFields(raw map[string]string) (fields Fields, err error) {
	fields = Fields{}
	for k, v := range raw {
		if strings.HasPrefix(k, "_") {
			continue
		}
		fields[k], err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return
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

// NoPersistRedisCounter  use redis implements Counter which not be pesisisted
type NoPersistRedisCounter struct {
	c.Initable
	Name        string
	redisClient *cache.RedisClient
	cacheParam  *cache.ParamConf
}

// NewNoPersistRedisCounter create new NewNoPersistRedisCounter
func NewNoPersistRedisCounter(name string, redisClient *cache.RedisClient, cacheParm *cache.ParamConf) (*NoPersistRedisCounter, error) {
	if c.HasNil(redisClient, cacheParm) {
		return nil, errors.New("invalid params")
	}
	return &NoPersistRedisCounter{
		Name:        name,
		redisClient: redisClient,
		cacheParam:  cacheParm,
	}, nil
}

// Init implements Initable.Init()
func (p *NoPersistRedisCounter) Init() error {
	if c.HasNil(p.redisClient, p.cacheParam) {
		return fmt.Errorf("RedisClient,cacheParam must not be nil")
	}
	return nil
}

// Incr implements Counter.Incr
func (p *NoPersistRedisCounter) Incr(counterID string, fieldAndDelta Fields) error {
	if counterID == "" || len(fieldAndDelta) == 0 {
		return errors.New("invalid params")
	}

	param := p.cacheParam.NewParamKey(counterID)
	pipeline, err := cache.NewPipeline(p.redisClient)
	if err != nil {
		return err
	}
	defer pipeline.Close()
	for k, v := range fieldAndDelta {
		if err := pipeline.Send(param, cache.HINCRBY, param.Key(), k, v); err != nil {
			return err
		}
	}
	if param.Expire() > 0 {
		if err := pipeline.Send(param, cache.EXPIRE, param.Key(), param.Expire()); err != nil {
			return err
		}
	}
	_, err = pipeline.Receive()
	if err != nil {
		return err
	}
	return nil
}

// Get implements Counter.Get
func (p *NoPersistRedisCounter) Get(counterID string) (fields Fields, err error) {
	if counterID == "" {
		return nil, errors.New("invalid params")
	}

	param := p.cacheParam.NewParamKey(counterID)
	reply, err := redis.Int64Map(p.redisClient.Do(param, func(conn redis.Conn) (interface{}, error) {
		return conn.Do(cache.HGETALL, param.Key())
	}))
	if err != nil {
		return nil, err
	}
	if len(reply) == 0 {
		return nil, nil
	}
	return Fields(reply), nil
}

// Del implements Counter.Del
func (p *NoPersistRedisCounter) Del(counterID string) error {
	if counterID == "" {
		return errors.New("invalid params")
	}

	param := p.cacheParam.NewParamKey(counterID)
	_, err := p.redisClient.Del(param)
	return err
}

// DelFields delete counter fileds of counterID
func (p *NoPersistRedisCounter) DelFields(counterID string, fields ...string) error {
	if counterID == "" || len(fields) == 0 {
		return errors.New("invalid params")
	}

	param := p.cacheParam.NewParamKey(counterID)
	args := []interface{}{param.Key()}
	args = append(args, (c.StringSlice(fields)).ToInterface()...)
	_, err := p.redisClient.Do(param, func(conn redis.Conn) (interface{}, error) {
		return conn.Do(cache.HDEL, args...)
	})
	return err
}
