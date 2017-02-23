package cache

import (
	"strconv"
	"testing"

	"github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/assert"
)

var (
	redisServer = &RedisServer{
		ID:   "test",
		Host: "127.0.0.1",
		Port: 6379,
	}
	r *RedisClient
)

func init() {
	err := redisServer.initPoolWithDefault()
	if err != nil {
		panic(err)
	}
	var groups = map[string][]*RedisServer{"test": []*RedisServer{redisServer}, "example": []*RedisServer{redisServer}}
	r = NewRedisClient(groups)
}

func TestRedis(t *testing.T) {

	param := NewParamConf("test", "test_", 0)
	testSetGet(t, r, param)

	expireParam := NewParamConf("test", "test_ex_", 20)
	testSetGet(t, r, expireParam)

	confKey := expireParam.NewParamKey("server")
	err := r.SetObject(confKey, redisServer)
	assert.Nil(t, err)

	server := RedisServer{}
	err = r.GetObject(confKey, &server)
	assert.Nil(t, err)
	assert.EqualValues(t, server.ID, redisServer.ID)

	ageParam := param.NewParamKey("age")
	exist, err := r.Exists(ageParam)
	assert.Nil(t, err)
	assert.True(t, exist)
	deleted, err := r.Del(ageParam)
	assert.Nil(t, err)
	assert.True(t, deleted)
	exist, err = r.Exists(ageParam)
	assert.Nil(t, err)
	assert.False(t, exist)

	ageNotExistParam := expireParam.NewParamKey("age_not_exist")
	expired, err := r.Expire(ageNotExistParam)
	assert.False(t, expired)
	deleted, err = r.Del(ageNotExistParam)
	assert.Nil(t, err)
	assert.False(t, deleted)
}

func testSetGet(t *testing.T, r *RedisClient, param *ParamConf) {
	ageParam := param.NewParamKey("age")
	assert.Nil(t, r.Set(ageParam, 10))
	reply, err := redis.Int(r.Get(ageParam))
	assert.Nil(t, err)
	assert.EqualValues(t, 10, reply)

	v, err := r.GetInt(ageParam)
	assert.Nil(t, err)
	assert.EqualValues(t, 10, v)
	ageNotExistParam := param.NewParamKey("age_not_exist")
	v, err = r.GetInt(ageNotExistParam)
	assert.NotNil(t, err)

	v64, err := r.GetInt64(ageParam)
	assert.Nil(t, err)
	assert.EqualValues(t, 10, v64)

	f64, err := r.GetFloat64(ageParam)
	assert.Nil(t, err)
	assert.EqualValues(t, 10, f64)

	s, err := r.GetString(ageParam)
	assert.Nil(t, err)
	assert.EqualValues(t, "10", s)
}

type TestUser struct {
	Name string
	Age  int
}

func TestPipeline(t *testing.T) {
	pipeline, err := NewPipeline(r)
	if err != nil {
		panic(err)
	}
	defer pipeline.Close()
	paramOdd := NewParamConf("test", "u_odd_", 0)
	paramEven := NewParamConf("test", "u_even_", 0)

	// set user
	for i := 0; i < 10; i++ {
		var paramConf *ParamConf
		if i%2 == 0 {
			paramConf = paramEven
		} else {
			paramConf = paramOdd
		}
		user := &TestUser{Name: "user" + strconv.Itoa(i)}
		param := paramConf.NewParamKey(strconv.Itoa(i))
		bytes, _ := MsgPackEncodeBytes(user)
		pipeline.Send(param, SET, param.Key(), bytes)
	}
	// get user
	for i := 0; i < 11; i++ {
		var paramConf *ParamConf
		if i%2 == 0 {
			paramConf = paramEven
		} else {
			paramConf = paramOdd
		}
		param := paramConf.NewParamKey(strconv.Itoa(i))
		pipeline.Send(param, GET, param.Key())
	}

	// del user
	for i := 0; i < 10; i++ {
		var paramConf *ParamConf
		if i%2 == 0 {
			paramConf = paramEven
		} else {
			paramConf = paramOdd
		}
		param := paramConf.NewParamKey(strconv.Itoa(i))
		pipeline.Send(param, DEL, param.Key())
	}

	replies, err := pipeline.Receive()
	assert.Nil(t, err)
	assert.Equal(t, 31, len(replies))

	setReplies := replies[0:10]
	for _, v := range setReplies {
		seted, err := redis.String(v.Reply, v.Err)
		assert.Nil(t, err)
		assert.Equal(t, ReplyOK, seted)
	}
	getReplies := replies[10:20]
	for i, v := range getReplies {
		bytes, err := redis.Bytes(v.Reply, v.Err)
		assert.Nil(t, err)
		user := &TestUser{}
		MsgPackDecodeBytes(bytes, user)
		assert.Equal(t, "user"+strconv.Itoa(i), user.Name)
	}

	getFailReply := replies[20]
	assert.Nil(t, getFailReply.Reply)
	assert.Nil(t, getFailReply.Err)

	delReplies := replies[21:31]
	for _, v := range delReplies {
		deleted, err := redis.Bool(v.Reply, v.Err)
		assert.Nil(t, err)
		assert.True(t, deleted)
	}
}

func TestGetObjects(t *testing.T) {
	paramConf := NewParamConf("test", "u__", 0)
	keys := []string{}

	// set user
	for i := 0; i < 10; i++ {
		k := strconv.Itoa(i)
		user := &TestUser{Name: "user" + strconv.Itoa(i), Age: i}
		param := paramConf.NewParamKey(k)
		bytes, _ := MsgPackEncodeBytes(user)
		r.Set(param, bytes)
		keys = append(keys, k)
	}

	keys = append([]string{"-1"}, keys...)
	var users []*TestUser
	err := r.GetObjects(paramConf, keys, &users)
	assert.Nil(t, err)
	assert.Equal(t, 11, len(users))
	assert.Nil(t, users[0])
	for i, v := range users[1:] {
		assert.Equal(t, i, v.Age)
		assert.Equal(t, "user"+strconv.Itoa(i), v.Name)
	}
}