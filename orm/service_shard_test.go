package orm

import (
	"database/sql"
	"testing"

	c "github.com/d0ngw/go/common"
	"github.com/stretchr/testify/assert"
)

func TestShardDBServcie(t *testing.T) {
	defaultMetaReg.clean()
	tm := tmodel{}
	AddMeta(&tm)
	user := &User{}
	AddMeta(user)

	conf := &shardConf{}
	err := c.LoadYAMLFromPath("testdata/shard.yaml", conf)
	assert.NoError(t, err)

	err = conf.Parse()
	assert.NoError(t, err)

	shardServcie := NewSimpleShardDBService(NewMySQLDBPool)
	shardServcie.DBShardConfig = conf
	shardServcie.EntityShardConfig = conf

	err = shardServcie.Init()
	assert.Nil(t, err)

	op, err := shardServcie.NewOp()
	assert.Equal(t, "test0", op.PoolName())

	var shardSvr ShardDBService = shardServcie
	assert.NotNil(t, shardSvr)

	op, err = shardServcie.NewOpByShardName("no exist")
	assert.NotNil(t, err)
	assert.Nil(t, op)

	op, err = shardServcie.NewOpByShardName("test0")
	assert.NotNil(t, op)
	assert.Nil(t, err)

	op, err = shardServcie.NewOpByEntity(&tm, "")
	assert.NotNil(t, op)
	assert.Nil(t, err)
	assert.Nil(t, tm.tblShardFunc)
	assert.Equal(t, "test0", op.PoolName())

	tm.ID = 2
	op, err = shardServcie.NewOpByEntity(&tm, "test_db_shard_hash")
	assert.NotNil(t, op)
	assert.Nil(t, err)
	assert.NotNil(t, tm.tblShardFunc)
	assert.Equal(t, "test_2", op.PoolName())

	tblShard, err := tm.tblShardFunc()
	assert.Nil(t, err)
	assert.Equal(t, "tt_2", tblShard)

	tm.Name = sql.NullString{String: "ok", Valid: true}
	err = Add(op, &tm)
	assert.Nil(t, err)

	ret, err := Get(op, &tm, tm.ID)
	assert.Nil(t, err)
	assert.NotNil(t, ret)

	del, err := Del(op, &tm, tm.ID)
	assert.Nil(t, err)
	assert.True(t, del)

	user0 := &User{
		Name: sql.NullString{String: "u0", Valid: true},
		Age:  1,
	}
	user1 := &User{
		Name: sql.NullString{String: "u1", Valid: true},
		Age:  2,
	}
	user2 := &User{
		Name: sql.NullString{String: "u2", Valid: true},
		Age:  3,
	}

	err = op.SetupTableShard(user0, "")
	assert.NoError(t, err)

	err = op.SetupTableShard(user1, "")
	assert.NoError(t, err)

	err = op.SetupTableShard(user2, "")
	assert.NoError(t, err)

	_, err = op.DoInTrans(func(tx *sql.Tx) (interface{}, error) {
		e := Add(op, &tm)
		assert.NoError(t, e)

		e = Add(op, user0)
		assert.NoError(t, e)

		e = Add(op, user1)
		assert.NoError(t, e)

		e = Add(op, user2)
		assert.NoError(t, e)
		return nil, nil
	})
}
