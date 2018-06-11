package common

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"runtime"

	"gopkg.in/yaml.v2"
)

var (
	errInvalidConf = errors.New("invalid conf")
)

// LoadYAMLFromPath 将YAML文件中的配置加载到到结构体target中
func LoadYAMLFromPath(filename string, target interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return LoadYAMl(data, target)
}

// LoadYAMl 将data中的YAML配置加载到到结构体target中
func LoadYAMl(data []byte, target interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("Can't load yaml config fomr emtpyt data")
	}
	return yaml.Unmarshal([]byte(data), target)
}

// Configurer 配置器
type Configurer interface {
	//解析配置
	Parse() error
}

// LogConfig 日志配置
type LogConfig struct {
	Conf string //日志的配置文件
}

// Parse 解析日志配置
func (p *LogConfig) Parse() error {
	if len(p.Conf) > 0 {
		InitLogger(p.Conf)
	}
	return nil
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

// LoadConfig 从configDir目录下的多个path指定的配置文件中加载配置
func LoadConfig(config Configurer, addonConfig string, configDir string, pathes ...string) (err error) {
	if len(pathes) == 0 && addonConfig == "" {
		return errInvalidConf
	}

	var content []byte
	if addonConfig != "" {
		content = append(content, addonConfig...)
		content = append(content, []byte("\n")...)
	}
	for _, p := range pathes {
		p = path.Join(configDir, p)
		Infof("load conf from:%s", p)
		cnt, err := ioutil.ReadFile(p)
		if err != nil {
			return err
		}
		if len(cnt) == 0 {
			Warnf("empty content in %s", p)
			continue
		}
		content = append(content, cnt...)
		content = append(content, []byte("\n")...)
	}
	err = LoadYAMl(content, config)
	if err != nil {
		return err
	}
	return
}
