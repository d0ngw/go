package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncDec(t *testing.T) {
	redisServer := &RedisServer{
		ID:   "test",
		Host: "127.0.0.1",
		Port: 6379,
	}
	bytes, err := MsgPackEncodeBytes(redisServer)
	assert.Nil(t, err)

	server := &RedisServer{}
	err = MsgPackDecodeBytes(bytes, server)
	assert.Nil(t, err)
	assert.Equal(t, *redisServer, *server)

	bytes, err = MsgPackEncodeBytes(nil)
	assert.Nil(t, err)

	var v *int
	err = MsgPackDecodeBytes(bytes, &v)
	assert.Nil(t, err)
	assert.Nil(t, v)

	servers := []*RedisServer{server, server}
	bytes, err = MsgPackEncodeBytes(servers)
	assert.Nil(t, err)
}
