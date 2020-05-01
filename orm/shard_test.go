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

func (p *shardConf) DBShardConfig() *DBShardConfig {
	return p.DBShards
}

func (p *shardConf) EntityShardConfig() *EntityShardConfig {
	return p.EntityShards
}

func TestShardConfig(t *testing.T) {
	defaultMetaReg.clean()
	tm := tmodel{}
	meta := AddMeta(&tm)
	AddMeta(&User{})

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

	dbHashRule := testDBShardHash.DBShard.Hash
	tableHasRule := testDBShardHash.TableShard.Hash
	assert.NotNil(t, dbHashRule)
	assert.NotNil(t, tableHasRule)

	var hashTest = func(hRule *HashRule, namePrefix string) {
		assert.NotNil(t, dbHashRule)
		assert.EqualValues(t, 100, hRule.Count)
		assert.Equal(t, namePrefix, hRule.NamePrefix)
		assert.Equal(t, "id", hRule.FieldName)
		name, err := hRule.Shard(0)
		assert.Nil(t, err)
		assert.Equal(t, namePrefix+"0", name)
		name, err = hRule.Shard(1)
		assert.Nil(t, err)
		assert.Equal(t, namePrefix+"1", name)
		name, err = hRule.Shard(99)
		assert.Nil(t, err)
		assert.Equal(t, namePrefix+"99", name)
	}
	hashTest(dbHashRule, "test_")
	hashTest(tableHasRule, "tt_")

	testDBShardNamed := entityRule.rules["test_db_shard_named"]
	assert.NotNil(t, testDBShardNamed)

	dbNamedRule := testDBShardNamed.DBShard.Named
	tableNamedRule := testDBShardNamed.TableShard.Named
	assert.NotNil(t, dbNamedRule)
	assert.NotNil(t, tableNamedRule)

	var namedTest = func(nRule *NamedRule, name string) {
		assert.Equal(t, name, nRule.Name)
		name, err := nRule.Shard(0)
		assert.Nil(t, err)
		assert.Equal(t, name, name)
	}
	namedTest(dbNamedRule, "test0")
	namedTest(tableNamedRule, "tt")

	testDBShardNumRange := entityRule.rules["test_db_shard_num_range"]
	assert.NotNil(t, testDBShardNumRange)

	dbNumRangeRule := testDBShardNumRange.DBShard.NumRange
	tableNumRangeRule := testDBShardNumRange.TableShard.NumRange
	assert.NotNil(t, dbNumRangeRule)
	assert.NotNil(t, tableNumRangeRule)

	var numRangeTest = func(nrRule *NumRangeRule, defaultName string, namePrefix string) {
		assert.NotNil(t, nrRule)
		assert.Equal(t, "id", nrRule.FieldName)
		assert.Equal(t, defaultName, nrRule.DefaultName)
		assert.EqualValues(t, 3, len(nrRule.Ranges))
		assert.EqualValues(t, 0, nrRule.Ranges[0].Begin)
		assert.EqualValues(t, 100, nrRule.Ranges[0].End)
		assert.EqualValues(t, 101, nrRule.Ranges[1].Begin)
		assert.EqualValues(t, 200, nrRule.Ranges[1].End)
		assert.EqualValues(t, 500, nrRule.Ranges[2].Begin)
		assert.EqualValues(t, 1000, nrRule.Ranges[2].End)

		var name string

		for i := -100; i <= -1; i++ {
			name, err = nrRule.Shard(i)
			assert.Nil(t, err)
			assert.Equal(t, defaultName, name, "i=%d", i)
		}

		for i := 0; i <= 100; i++ {
			name, err = nrRule.Shard(i)
			assert.Nil(t, err)
			assert.Equal(t, namePrefix+"100", name, "i=%d", i)
		}

		for i := 101; i <= 200; i++ {
			name, err = nrRule.Shard(i)
			assert.Nil(t, err)
			assert.Equal(t, namePrefix+"200", name, "i=%d", i)
		}

		for i := 201; i <= 499; i++ {
			name, err = nrRule.Shard(i)
			assert.Nil(t, err)
			assert.Equal(t, defaultName, name, "i=%d", i)
		}

		for i := 500; i <= 1000; i++ {
			name, err = nrRule.Shard(i)
			assert.Nil(t, err)
			assert.Equal(t, namePrefix+"1000", name, "i=%d", i)
		}

		for i := 1001; i <= 10000; i++ {
			name, err = nrRule.Shard(i)
			assert.Nil(t, err)
			assert.Equal(t, defaultName, name, "i=%d", i)
		}

		for i := 100001; i <= 100005; i++ {
			name, err = nrRule.Shard(i)
			assert.Nil(t, err)
			assert.Equal(t, defaultName, name, "i=%d", i)
		}
	}

	numRangeTest(dbNumRangeRule, "test0", "test_")
	numRangeTest(tableNumRangeRule, "tt", "tt_")
	defaultMetaReg.clean()
}
