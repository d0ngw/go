package counter

import (
	"fmt"
	"io/ioutil"

	"github.com/garyburd/redigo/redis"
)

// Scripts define persist counter lua scripts
type Scripts struct {
	Update  string `yaml:"update_lua"`
	SetSync string `yaml:"sync_set_lua"`
	Evict   string `yaml:"evict_lua"`
	HgetAll string `yaml:"hgetall_lua"`
	Del     string `yaml:"del_lua"`
	update  *redis.Script
	setSync *redis.Script
	evict   *redis.Script
	hgetAll *redis.Script
	del     *redis.Script
}

// Lua
const (
	LUAFALSE int = 0
	LUATRUE  int = 1
)

// Init implements Init
func (p *Scripts) Init() (err error) {
	scripts := []struct {
		path string
		dest **redis.Script
	}{
		{p.Update, &p.update},
		{p.SetSync, &p.setSync},
		{p.Evict, &p.evict},
		{p.HgetAll, &p.hgetAll},
		{p.Del, &p.del},
	}

	for _, v := range scripts {
		if err := p.loadScriptFromFile(v.path, v.dest); err != nil {
			return err
		}
	}
	return nil
}

func (p *Scripts) loadScriptFromFile(luaPath string, dest **redis.Script) error {
	data, err := ioutil.ReadFile(luaPath)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("empty lua script in %s", luaPath)
	}
	script := redis.NewScript(1, string(data))
	*dest = script
	return nil
}
