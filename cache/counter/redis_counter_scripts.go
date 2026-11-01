package counter

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/gomodule/redigo/redis"
)

// Scripts define persist counter lua scripts
type Scripts struct {
	Update         string `yaml:"update_lua"`
	SetSync        string `yaml:"sync_set_lua"`
	Evict          string `yaml:"evict_lua"`
	HgetAll        string `yaml:"hgetall_lua"`
	Del            string `yaml:"del_lua"`
	loadFromString bool
	update         *redis.Script
	setSync        *redis.Script
	evict          *redis.Script
	hgetAll        *redis.Script
	del            *redis.Script
}

//go:embed lua/update.lua
var defaultUpdateLua string

//go:embed lua/set_sync.lua
var defaultSetSyncLua string

//go:embed lua/evict.lua
var defaultEvictLua string

//go:embed lua/get_all.lua
var defaultGetAllLua string

//go:embed lua/delete.lua
var defaultDeleteLua string

//go:embed lua/snapshot.lua
var counterSnapshotLua string

var counterSnapshotScript = redis.NewScript(1, counterSnapshotLua)

// DefaultScripts returns an initialized, independent set of embedded scripts.
func DefaultScripts() *Scripts {
	scripts := &Scripts{
		Update: defaultUpdateLua, SetSync: defaultSetSyncLua,
		Evict: defaultEvictLua, HgetAll: defaultGetAllLua, Del: defaultDeleteLua,
		loadFromString: true,
	}
	// Embedded sources are nonempty and validated by integration tests.
	if err := scripts.Init(); err != nil {
		panic(err)
	}
	return scripts
}

// NewScripts configures custom scripts loaded from strings or files.
func NewScripts(loadFromString bool) *Scripts {
	return &Scripts{loadFromString: loadFromString}
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
		if p.loadFromString {
			if err := p.loadScriptFromString(v.path, v.dest); err != nil {
				return err
			}
		} else {
			if err := p.loadScriptFromFile(v.path, v.dest); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Scripts) loadScriptFromFile(luaPath string, dest **redis.Script) error {
	data, err := os.ReadFile(luaPath)
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

func (p *Scripts) loadScriptFromString(luaData string, dest **redis.Script) error {
	if len(luaData) == 0 {
		return errors.New("empty Lua script")
	}
	script := redis.NewScript(1, string(luaData))
	*dest = script
	return nil
}
