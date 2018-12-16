package inject

import (
	"os"
	"path"
	"sync"

	c "github.com/d0ngw/go/common"
)

var (
	// Module 声明了通用的服务
	module      = NewModule()
	confs       = []string{"common.yaml"}
	injectMutex sync.Mutex
	injectorEnv = map[string]*Injector{}
)

// SetupInjector 从env指定的环境配置初始化配置,构建Injector
func SetupInjector(config c.Configurer, addonConfig string, env string, modules ...*Module) (*Injector, error) {
	injectMutex.Lock()
	defer injectMutex.Unlock()

	injector, ok := injectorEnv[env]
	if ok {
		return injector, nil
	}

	workfDir := os.Getenv(c.EnvWorkfDir)
	if workfDir != "" {
		err := os.Chdir(workfDir)
		if err != nil {
			c.Errorf("can't change work dir to %s", workfDir)
			return nil, err
		}
	}

	c.Infof("work dir:%s", workfDir)
	mainConf := "conf_" + env + ".yaml"
	allConfs := []string{mainConf}
	allConfs = append(allConfs, confs...)

	existConfs := []string{}
	//check
	for _, confFile := range allConfs {
		if _, err := os.Stat(path.Join("conf", confFile)); os.IsNotExist(err) {
			c.Warnf("config %s does not exist,skip", confFile)
		} else {
			existConfs = append(existConfs, confFile)
		}
	}

	err := c.LoadConfig(config, addonConfig, path.Join("conf"), existConfs...)
	if err != nil {
		return nil, err
	}
	err = config.Parse()
	if err != nil {
		return nil, err
	}

	// 绑定核心的服务
	module.Bind(config)
	var allModuls []*Module
	allModuls = append(allModuls, module)
	allModuls = append(allModuls, modules...)
	injector = NewInjector(allModuls)
	err = injector.Initialize()
	if err != nil {
		return nil, err
	}
	injectorEnv[env] = injector
	return injector, nil
}

// Injected 判断是否已经完成注入
type Injected interface {
	// 是否已经完成注入
	IsInjected() bool
}

// IsInjected 判断i是否实现了Injected接口
// 当i实现了Injected接口时,ok为true,这时injected表示接口i是否已经完成了注入
func IsInjected(i interface{}) (ok bool, injected bool) {
	if i == nil {
		return
	}
	injectedi, ok := i.(Injected)
	if ok {
		injected = injectedi.IsInjected()
	}
	return
}
