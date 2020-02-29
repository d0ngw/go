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
	injectMutex sync.Mutex
	injectorEnv = map[string]*Injector{}
)

// SetupInjector 从env指定的环境配置初始化配置,构建Injector
func SetupInjector(config c.Configurer, addonConfig string, env string, modules ...*Module) (*Injector, error) {
	return SetupInjectorWithLoader(c.FileLoader, config, addonConfig, env, modules...)
}

// SetupInjectorWithLoader 从env指定的环境配置初始化配置,构建Injector
func SetupInjectorWithLoader(loader c.ConfigLoader, config c.Configurer, addonConfig string, env string, modules ...*Module) (*Injector, error) {
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

	confDir := "conf"
	var (
		confs = []string{"common.yaml"}
	)

	if env != "" {
		confs = append(confs, "conf_"+env+".yaml")
	} else {
		confs = append(confs, "conf.yaml")
	}

	for _, f := range confs {
		conf := path.Join(confDir, f)
		if exist, err := loader.Exist(conf); err != nil {
			c.Errorf("check %s fail, err:%v", conf, err)
			return nil, err
		} else if !exist {
			c.Warnf("%s doesn't exist, skip", conf)
		} else {
			if content, err := loader.Load(conf); err != nil {
				c.Errorf("load %s fail,err:%v", conf, err)
				return nil, err
			} else if len(content) > 0 {
				addonConfig += "\n" + string(content) + "\n"
			}
		}
	}

	err := c.LoadConfigWithLoader(loader, config, addonConfig, confDir)
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
