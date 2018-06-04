package cache

import (
	"errors"
	"fmt"
	"reflect"

	c "github.com/d0ngw/go/common"
	"github.com/garyburd/redigo/redis"
)

// redis的命令及响应
const (
	ReplyOK          = "OK"
	DEL              = "DEL"
	EXISTS           = "EXISTS"
	EXPIRE           = "EXPIRE"
	GET              = "GET"
	HDEL             = "HDEL"
	HGETALL          = "HGETALL"
	HINCRBY          = "HINCRBY"
	SET              = "SET"
	SETEX            = "SETEX"
	ZADD             = "ZADD"
	ZCARD            = "ZCARD"
	ZRANGE           = "ZRANGE"
	ZREM             = "ZREM"
	ZRANGEWITHSCORES = "ZRANGEWITHSCORES"
)

// RedisClient redis client
type RedisClient struct {
	groups map[string][]*RedisServer
}

// NewRedisClient create new Redis
func NewRedisClient(groups map[string][]*RedisServer) *RedisClient {
	return &RedisClient{groups: groups}
}

// NewRedisClientWithConf create redis from conf
func NewRedisClientWithConf(conf *RedisConf) *RedisClient {
	return &RedisClient{groups: conf.groups}
}

func (p *RedisClient) getServerIndex(param Param, servers []*RedisServer) (index int, err error) {
	serverCount := len(servers)
	if serverCount == 0 {
		return 0, fmt.Errorf("no servers for group %s", param.Group())
	}
	if serverCount == 1 {
		return 0, nil
	}
	hashCode := c.Fnv32Hashcode(param.Key())
	return hashCode % serverCount, nil
}

// GetGroupServers query the servers for group
func (p *RedisClient) GetGroupServers(group string) ([]*RedisServer, error) {
	if servers, ok := p.groups[group]; ok {
		return servers, nil
	}
	return nil, fmt.Errorf("can't find group for %s", group)
}

// GetConn acquire redis.Conn in param.Group.
// If has mutiple servers in redis group,choose server with hash code which generated by fnv(key) % len(servers)
func (p *RedisClient) GetConn(param Param) (conn redis.Conn, err error) {
	if param.Group() == "" || param.Key() == "" {
		return nil, fmt.Errorf("invalid params,groupId and key must not be empty")
	}
	if servers, ok := p.groups[param.Group()]; ok {
		serverIndex, err := p.getServerIndex(param, servers)
		if err != nil {
			return nil, err
		}
		return servers[serverIndex].pool.Get(), nil
	}
	return nil, fmt.Errorf("can't find redis group %s", param.Group())
}

// Do exec redis commands with param and key
func (p *RedisClient) Do(param Param, fn func(conn redis.Conn) (interface{}, error)) (reply interface{}, err error) {
	conn, err := p.GetConn(param)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	return fn(conn)
}

// Set param.Key() with value `data`,if param.Exipre >0,then set key with expire second
func (p *RedisClient) Set(param Param, data interface{}) error {
	reply, err := p.Do(param, func(conn redis.Conn) (reply interface{}, err error) {
		if param.Expire() > 0 {
			reply, err = conn.Do(SET, param.Key(), data, "EX", param.Expire())
		} else {
			reply, err = conn.Do(SET, param.Key(), data)
		}
		return
	})
	reply, err = redis.String(reply, err)
	if err != nil {
		return err
	}
	if reply != ReplyOK {
		return fmt.Errorf("reply error:%s", reply)
	}
	return nil
}

// Get value from redis with param and key,if the param.Expire >0 then will EXPIRE the key
func (p *RedisClient) Get(param Param) (reply interface{}, ok bool, err error) {
	reply, err = p.Do(param, func(conn redis.Conn) (interface{}, error) {
		if param.Expire() > 0 {
			if err := conn.Send(GET, param.Key()); err != nil {
				fmt.Printf("send cmd fail,err:%s", err)
				return nil, err
			}
			if err := conn.Send(EXPIRE, param.Key(), param.Expire()); err != nil {
				fmt.Printf("send cmd fail,err:%s", err)
				return nil, err
			}
			if err := conn.Flush(); err != nil {
				fmt.Printf("flush cmd fail,err:%s", err)
				return nil, err
			}
			r, err := conn.Receive()
			conn.Receive() //ignore expire
			return r, err
		}
		return conn.Do(GET, param.Key())
	})

	if reply != nil {
		ok = true
	}
	return
}

// GetInt get int value from redis with param
func (p *RedisClient) GetInt(param Param) (reply int, ok bool, err error) {
	r, ok, err := p.Get(param)
	if !ok {
		return
	}
	reply, _ = redis.Int(r, err)
	return
}

// GetInt64 get int64 value from redis with param
func (p *RedisClient) GetInt64(param Param) (reply int64, ok bool, err error) {
	r, ok, err := p.Get(param)
	if !ok {
		return
	}
	reply, _ = redis.Int64(r, err)
	return
}

// GetFloat64 get int64 value from redis with param
func (p *RedisClient) GetFloat64(param Param) (reply float64, ok bool, err error) {
	r, ok, err := p.Get(param)
	if !ok {
		return
	}
	reply, _ = redis.Float64(r, err)
	return
}

// GetString get int64 value from redis with param
func (p *RedisClient) GetString(param Param) (reply string, ok bool, err error) {
	r, ok, err := p.Get(param)
	if !ok {
		return
	}
	reply, _ = redis.String(r, err)
	return
}

// SetObject set param.Key() with value `data`,if param.Exipre >0,then set key with expire second
func (p *RedisClient) SetObject(param Param, data interface{}) error {
	bytes, err := MsgPackEncodeBytes(data)
	if err != nil {
		return err
	}
	return p.Set(param, bytes)
}

// GetObject get bytes whose key is param.Key(),then decode bytes to dest
func (p *RedisClient) GetObject(param Param, dest interface{}) (ok bool, err error) {
	r, ok, err := p.Get(param)
	if !ok {
		return
	}
	reply, _ := redis.Bytes(r, err)
	err = MsgPackDecodeBytes(reply, dest)
	return
}

// GetObjects batch get struct object,use MsgPackDecodeBytes to decode bytes and append  to dest
func (p *RedisClient) GetObjects(paramConf *ParamConf, keys []string, dest interface{}, getByKey func(key string, index int) (interface{}, error)) error {
	if len(keys) == 0 {
		return fmt.Errorf("not allow empty keys")
	}

	val, _, typ := c.ExtractRefTuple(dest)
	if typ.Kind() != reflect.Slice {
		return fmt.Errorf("dest must be pointer of slice")
	}

	if val.Len() != len(keys) {
		return fmt.Errorf("the length of keys %d != dest length %d", len(keys), val.Len())
	}

	valElemTyp := reflect.TypeOf(val.Index(0).Interface())
	if valElemTyp.Kind() != reflect.Ptr || valElemTyp.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest element must be pointer of struct")
	}

	pipeline, _ := NewPipeline(p)

	defer pipeline.Close()
	for _, k := range keys {
		param := paramConf.NewParamKey(k)
		if err := pipeline.Send(param, GET, param.Key()); err != nil {
			return err
		}
	}

	replies, err := pipeline.Receive()
	if err != nil {
		return err
	}

	elemTyp := typ.Elem()

	for i, reply := range replies {
		if reply.Err != nil {
			return reply.Err
		}
		valElement := val.Index(i)

		if bytes, _ := redis.Bytes(reply.Reply, err); bytes != nil {
			err = MsgPackDecodeBytes(bytes, valElement.Interface())
			if err != nil {
				return err
			}
		} else {
			var found bool
			if getByKey != nil {
				ret, err := getByKey(keys[i], i)
				if err != nil {
					return err
				}
				if ret != nil {
					valElement.Set(reflect.ValueOf(ret))
					found = true
				}
			}
			if !found {
				var zero = reflect.Zero(elemTyp)
				valElement.Set(zero)
			}
		}
	}
	return nil
}

// Del del the param.Key()
func (p *RedisClient) Del(param Param) (deleted bool, err error) {
	deleted, err = redis.Bool(p.Do(param, func(conn redis.Conn) (reply interface{}, err error) {
		return conn.Do(DEL, param.Key())
	}))
	return
}

// Exists check the param.Key() exist
func (p *RedisClient) Exists(param Param) (exists bool, err error) {
	exists, err = redis.Bool(p.Do(param, func(conn redis.Conn) (reply interface{}, err error) {
		return conn.Do(EXISTS, param.Key())
	}))
	return
}

// Expire set timeout on key `param.Key()`,the timeout is `param.Expire()` second
func (p *RedisClient) Expire(param Param) (expired bool, err error) {
	expired, err = redis.Bool(p.Do(param, func(conn redis.Conn) (reply interface{}, err error) {
		return conn.Do(EXPIRE, param.Key(), param.Expire())
	}))
	return
}

// Eval lua script for param.Key() with args
func (p *RedisClient) Eval(param Param, script *redis.Script, args ...interface{}) (reply interface{}, err error) {
	if c.HasNil(param, script) {
		return nil, fmt.Errorf("invalid params")
	}
	keyAndArgs := []interface{}{param.Key()}
	keyAndArgs = append(keyAndArgs, args...)
	conn, err := p.GetConn(param)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return script.Do(conn, keyAndArgs...)
}

// Pipeline the command and results
type Pipeline struct {
	r           *RedisClient
	groupConns  map[string]map[int]redis.Conn
	usedConns   []redis.Conn
	resultConns []redis.Conn
}

// NewPipeline new pipeline from RedisClient
func NewPipeline(r *RedisClient) (*Pipeline, error) {
	if r == nil {
		return nil, fmt.Errorf("nil RedisClient")
	}
	return &Pipeline{
		r:          r,
		groupConns: map[string]map[int]redis.Conn{},
	}, nil
}

// PipelineReply reply from pipeline
type PipelineReply struct {
	Reply interface{}
	Err   error
}

// Send write the command to the redis conn out buffer.
func (p *Pipeline) Send(param Param, command string, args ...interface{}) error {
	r := p.r
	servers, ok := r.groups[param.Group()]
	if !ok {
		return fmt.Errorf("Not found group %s", param.Group())
	}

	conns := p.groupConns[param.Group()]
	if conns == nil {
		conns = map[int]redis.Conn{}
		p.groupConns[param.Group()] = conns
	}

	serverIndex, err := r.getServerIndex(param, servers)
	if err != nil {
		return err
	}

	conn, ok := conns[serverIndex]
	if !ok {
		conn = servers[serverIndex].pool.Get()
		conns[serverIndex] = conn
		p.usedConns = append(p.usedConns, conn)
	}
	err = conn.Send(command, args...)
	if err != nil {
		return err
	}
	p.resultConns = append(p.resultConns, conn)
	return nil
}

// Flush all command in the output buffers.
func (p *Pipeline) Flush() error {
	for _, conn := range p.usedConns {
		if err := conn.Flush(); err != nil {
			return err
		}
	}
	return nil
}

// Receive all reply from reids server
func (p *Pipeline) Receive() (replies []*PipelineReply, err error) {
	if err := p.Flush(); err != nil {
		return nil, err
	}

	for _, conn := range p.resultConns {
		reply := &PipelineReply{}
		reply.Reply, reply.Err = conn.Receive()
		replies = append(replies, reply)
	}
	return
}

// Close all redis connection
func (p *Pipeline) Close() {
	for _, conn := range p.usedConns {
		conn.Close()
	}
}

var luaLock = `
local lock_key = KEYS[1]
local lock_second = tonumber(ARGV[1])

local exist = redis.call("EXISTS", lock_key)
local locked = 0
if exist == 0 then
    redis.call("SETEX", lock_key, lock_second, 1)
    locked = 1
end
return { locked }
`

var lockScript = redis.NewScript(1, luaLock)

// TryLock try to lock lockKey in lockSencods
func TryLock(lockKey string, lockSencods int, paramConf *ParamConf, redisClient *RedisClient) (bool, error) {
	if lockKey == "" || lockSencods <= 0 || paramConf == nil || redisClient == nil {
		return false, errors.New("invalid params")
	}

	key := paramConf.NewParamKey(lockKey)
	conn, err := redisClient.GetConn(key)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	reply, err := redis.Ints(lockScript.Do(conn, key.Key(), lockSencods))
	if err != nil || len(reply) == 0 {
		return false, err
	}
	return reply[0] == 1, nil
}

// UnLock unlock lockey
func UnLock(lockKey string, paramConf *ParamConf, redisClient *RedisClient) error {
	if lockKey == "" || paramConf == nil || redisClient == nil {
		return errors.New("invalid params")
	}

	key := paramConf.NewParamKey(lockKey)
	_, err := redisClient.Del(key)
	return err
}

// CheckNilErr check the err:
//	if err == nil, return true,nil; if the err is ErrNIl return false,nil;,otherwise return false ,err
func CheckNilErr(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	if err == redis.ErrNil {
		return false, nil
	}
	return false, err
}
