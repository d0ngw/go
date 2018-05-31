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
	AddMeta(&tm)

	conf := &shardConf{}
	err := c.LoadYAMLFromPath("testdata/shard.yaml", conf)
	assert.NoError(t, err)

	err = conf.Parse()
	assert.NoError(t, err)

	defaultMetaReg.clean()
}
