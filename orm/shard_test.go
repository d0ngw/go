package orm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	c "github.com/d0ngw/go/common"
)

type shardConf struct {
	c.AppConfig
	DBShards     *DBShardConfig     `yaml:"db_shards"`
	EntityShards *EntityShardConfig `yaml:"entity_shards"`
}

func (p *shardConf) Parse() error {
	if err := p.AppConfig.Parse(); err != nil {
		return err
	}

	if err := p.DBShards.Parse(); err != nil {
		return err
	}

	if err := p.EntityShards.Parse(); err != nil {
		return err
	}
	return nil
}

func TestShardConfig(t *testing.T) {
	defaultMetaReg.clean()
	tm := tmodel{}
	meta := AddMeta(&tm)

	conf := &shardConf{}
	err := c.LoadYAMLFromPath("testdata/shard.yaml", conf)
	assert.NoError(t, err)

	err = conf.Parse()
	assert.NoError(t, err)

	assert.NotNil(t, conf.DBShards)
	assert.NotNil(t, conf.EntityShards)

	testShard := conf.DBShards.Shards["test0"]
	defaultShard := conf.DBShards.Default
	assert.NotNil(t, testShard)
	assert.NotNil(t, defaultShard)
	assert.True(t, defaultShard == "test0")

	entityRule := conf.EntityShards.entities[meta.Type().PkgPath()][meta.Type().Name()]
	assert.NotNil(t, entityRule)
	assert.NotNil(t, entityRule.meta)
	assert.True(t, meta == entityRule.meta)

	defaultRule := entityRule.defaultRule
	assert.NotNil(t, defaultRule)
	assert.Equal(t, defaultRule.Name, "default")
	assert.Equal(t, defaultRule, entityRule.rules["default"])

	testDBShardHash := entityRule.rules["test_db_shard_hash"]
	assert.NotNil(t, testDBShardHash)
	assert.NotNil(t, testDBShardHash.DBShard.Hash)
	assert.Equal(t, 100, testDBShardHash.DBShard.Hash.Count)
	assert.Equal(t, "tt_", testDBShardHash.DBShard.Hash.NamePrefix)
	assert.Equal(t, "id", testDBShardHash.DBShard.Hash.FieldName)
	name, err := testDBShardHash.DBShard.Hash.Shard(0)
	assert.Nil(t, err)
	assert.Equal(t, "tt_0", name)
	name, err = testDBShardHash.DBShard.Hash.Shard(1)
	assert.Nil(t, err)
	assert.Equal(t, "tt_1", name)
	name, err = testDBShardHash.DBShard.Hash.Shard(99)
	assert.Nil(t, err)
	assert.Equal(t, "tt_99", name)

	testDBShardNamed := entityRule.rules["test_db_shard_named"]
	assert.NotNil(t, testDBShardNamed)
	assert.NotNil(t, testDBShardNamed.DBShard.Named)
	assert.Equal(t, "tt", testDBShardNamed.DBShard.Named.Name)
	name, err = testDBShardNamed.DBShard.Named.Shard(0)
	assert.Nil(t, err)
	assert.Equal(t, "tt", name)

	testDBShardNumRange := entityRule.rules["test_db_shard_num_range"]
	assert.NotNil(t, testDBShardNumRange)
	assert.NotNil(t, testDBShardNumRange.DBShard.NumRange)
	assert.Equal(t, "id", testDBShardNumRange.DBShard.NumRange.FieldName)
	assert.Equal(t, "tt", testDBShardNumRange.DBShard.NumRange.DefaultName)
	assert.Equal(t, 3, len(testDBShardNumRange.DBShard.NumRange.Ranges))
	assert.Equal(t, 0, testDBShardNumRange.DBShard.NumRange.Ranges[0].Begin)
	assert.Equal(t, 100, testDBShardNumRange.DBShard.NumRange.Ranges[0].End)
	assert.Equal(t, 101, testDBShardNumRange.DBShard.NumRange.Ranges[1].Begin)
	assert.Equal(t, 200, testDBShardNumRange.DBShard.NumRange.Ranges[1].End)
	assert.Equal(t, 500, testDBShardNumRange.DBShard.NumRange.Ranges[2].Begin)
	assert.Equal(t, 1000, testDBShardNumRange.DBShard.NumRange.Ranges[2].End)

	for i := -100; i <= -1; i++ {
		name, err = testDBShardNumRange.DBShard.NumRange.Shard(i)
		assert.Nil(t, err)
		assert.Equal(t, "tt", name, "i=%d", i)
	}

	for i := 0; i <= 100; i++ {
		name, err = testDBShardNumRange.DBShard.NumRange.Shard(i)
		assert.Nil(t, err)
		assert.Equal(t, "tt_100", name, "i=%d", i)
	}

	for i := 101; i <= 200; i++ {
		name, err = testDBShardNumRange.DBShard.NumRange.Shard(i)
		assert.Nil(t, err)
		assert.Equal(t, "tt_200", name, "i=%d", i)
	}

	for i := 201; i <= 499; i++ {
		name, err = testDBShardNumRange.DBShard.NumRange.Shard(i)
		assert.Nil(t, err)
		assert.Equal(t, "tt", name, "i=%d", i)
	}

	for i := 500; i <= 1000; i++ {
		name, err = testDBShardNumRange.DBShard.NumRange.Shard(i)
		assert.Nil(t, err)
		assert.Equal(t, "tt_1000", name, "i=%d", i)
	}

	for i := 1001; i <= 10000; i++ {
		name, err = testDBShardNumRange.DBShard.NumRange.Shard(i)
		assert.Nil(t, err)
		assert.Equal(t, "tt", name, "i=%d", i)
	}

	for i := 100001; i <= 100005; i++ {
		name, err = testDBShardNumRange.DBShard.NumRange.Shard(i)
		assert.Nil(t, err)
		assert.Equal(t, "tt", name, "i=%d", i)
	}

	defaultMetaReg.clean()
}
