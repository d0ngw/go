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
	err = injector.Init()
	if err != nil {
		return nil, err
	}
	injectorEnv[env] = injector
	return injector, nil
}
