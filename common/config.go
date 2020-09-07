package common

import (
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
)

var (
	errInvalidConf = errors.New("invalid conf")
)

// ConfigLoader 配置内容加载器
type ConfigLoader interface {
	Load(configPath string) (content []byte, err error)

	Exist(configPath string) (exist bool, err error)
}

// ConfigFileLoader 从本地文件中加载配置
type ConfigFileLoader struct {
}

// Load impls ConfigLoader.Load
func (p *ConfigFileLoader) Load(configPath string) (content []byte, err error) {
	content, err = ioutil.ReadFile(configPath)
	return
}

// Exist impls ConfigLoader.Exist
func (p *ConfigFileLoader) Exist(configPath string) (exist bool, err error) {
	info, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		err = nil
		return
	}
	if info != nil {
		exist = !info.IsDir()
	}
	return
}

var (
	//FileLoader 默认加载
	FileLoader ConfigLoader = &ConfigFileLoader{}
)

// Configurer 配置器
type Configurer interface {
	//解析配置
	Parse() error
}

// LogConfig 日志配置
type LogConfig struct {
	Env        string `yaml:"env"`
	FileName   string `yaml:"file_name"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	NoCaller   bool   `yaml:"no_caller"`
	Level      string `yaml:"level"`
}

// LogConfiger the log configer
type LogConfiger interface {
	GetLogConfig() *LogConfig
}

// Parse 解析日志配置
func (p *LogConfig) Parse() error {
	return initLogger(p)
}

// RuntimeConfig 运行期配置
type RuntimeConfig struct {
	Maxprocs int //最大的PROCS个数
}

// Parse 解析运行期配置
func (p *RuntimeConfig) Parse() error {
	if p.Maxprocs > 0 {
		preProcs := runtime.GOMAXPROCS(p.Maxprocs)
		Infof("Set runtime.MAXPROCS to %v,old is %v", p.Maxprocs, preProcs)
	}
	return nil
}

// AppConfig 基础的应用配置
type AppConfig struct {
	*LogConfig          `yaml:"log"`
	*RuntimeConfig      `yaml:"runtime"`
	*ValidateRuleConfig `yaml:"validates"`
}

// Parse 解析基础的应用配置
func (p *AppConfig) Parse() error {
	return Parse(p)
}

// GetValidateRuleConfig implements ValidateConfiguer
func (p *AppConfig) GetValidateRuleConfig() *ValidateRuleConfig {
	return p.ValidateRuleConfig
}

// GetLogConfig impls LogConfiger
func (p *AppConfig) GetLogConfig() *LogConfig {
	return p.LogConfig
}

// Parse 解析配置
func Parse(conf interface{}) error {
	config := reflect.Indirect(reflect.ValueOf(conf))
	fieldCount := config.NumField()

	for i := 0; i < fieldCount; i++ {
		val := reflect.Indirect(config.Field(i))
		if !val.IsValid() {
			continue
		}

		if configFieldValue, ok := val.Addr().Interface().(Configurer); ok {
			if err := configFieldValue.Parse(); err != nil {
				return err
			}
		}
	}
	return nil
}
