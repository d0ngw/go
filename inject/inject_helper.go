package inject

import (
	"os"
	"path"

	c "github.com/d0ngw/go/common"
)

// ConfigModuler get module depends on the config
type ConfigModuler interface {
	ConfModule() (module *Module, err error)
}

// SetupInjector 从env指定的环境配置初始化配置,构建Injector
func SetupInjector(config c.Configurer, addonConfig string, env string, modules ...*Module) (*Injector, error) {
	return SetupInjectorWithLoader(c.FileLoader, config, addonConfig, env, modules...)
}

// EnvConfRoot conf root
const EnvConfRoot = "conf_root"

// SetupInjectorWithLoader 从env指定的环境配置初始化配置,构建Injector
func SetupInjectorWithLoader(loader c.ConfigLoader, config c.Configurer, addonConfig string, env string, modules ...*Module) (*Injector, error) {
	confDir := "conf"
	if os.Getenv(EnvConfRoot) != "" {
		confDir = os.Getenv(EnvConfRoot)
	}
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
			c.Infof("load conf from %s ", conf)
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

	var confModule *Module
	if configModuler, ok := config.(ConfigModuler); ok {
		if confModule, err = configModuler.ConfModule(); err != nil {
			return nil, err
		}
	}

	// 绑定核心的服务
	module := NewModule()
	module.Bind(config)
	var allModuls []*Module
	allModuls = append(allModuls, module)
	allModuls = append(allModuls, modules...)
	if confModule != nil {
		allModuls = append(allModuls, confModule)
	}
	injector := NewInjector(allModuls)
	err = injector.Initialize()
	if err != nil {
		return nil, err
	}
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
