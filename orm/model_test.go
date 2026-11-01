package orm

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReflect(t *testing.T) {
	var err error
	tm := tmodel{}
	defaultMetaReg.clean()

	_, err = defaultMetaReg.regModel(&tm)
	assert.NoError(t, err)
}

func TestAddMeta(t *testing.T) {
	defaultMetaReg.clean()
	tm := tmodel{}
	_meta := AddMeta(&tm)
	assert.Equal(t, "github.com/d0ngw/go/orm.tmodel", _meta.Name())
}

func TestShardEntity(t *testing.T) {
	tm := &tmodel{}
	tm.Name = sql.NullString{String: "abc", Valid: true}

	var entity Entity = tm

	shardEntity, ok := entity.(ShardEntity)
	assert.True(t, ok)
	assert.Equal(t, entity, shardEntity)
	assert.True(t, entity == shardEntity)
}
