package common

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
	"runtime"
)

//YAML配置

//将YAML文件中的配置加载到到结构体target中
func LoadYAMLFromPath(filename string, target interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return LoadYAMl(data, target)
}

//将data中的YAML配置加载到到结构体target中
func LoadYAMl(data []byte, target interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("Can't load yaml config fomr emtpyt data")
	}
	return yaml.Unmarshal([]byte(data), target)
}

//配置器
type Configurer interface {
	//解析配置
	Parse() error
}

//日志配置
type LogConfig struct {
	Conf string //日志的配置文件
}

func (self *LogConfig) Parse() error {
	if len(self.Conf) > 0 {
		InitLogger(self.Conf)
	}
	return nil
}

//运行期配置
type RuntimeConfig struct {
	Maxprocs int //最大的PROCS个数
}

func (self *RuntimeConfig) Parse() error {
	if self.Maxprocs > 0 {
		preProcs := runtime.GOMAXPROCS(self.Maxprocs)
		Infof("Set runtime.MAXPROCS to %v,old is %v", self.Maxprocs, preProcs)
	}
	return nil
}

//基础的应用配置
type AppConfig struct {
	LogConfig          `yaml:"log"`
	RuntimeConfig      `yaml:"runtime"`
	ValidateRuleConfig `yaml:"validates"`
}

//解析基础的应用配置
func (self *AppConfig) Parse() error {
	return Parse(self)
}

//解析配置
func Parse(conf interface{}) error {
	config := reflect.Indirect(reflect.ValueOf(conf))
	fieldCount := config.NumField()
	typ := config.Type()

	for i := 0; i < fieldCount; i++ {
		configField := config.Field(i)
		typField := typ.Field(i)
		Debugf("Found %#v", typField.Name)
		if configFieldValue, ok := configField.Addr().Interface().(Configurer); ok {
			Debugf("Parse %#v", configFieldValue)
			if err := configFieldValue.Parse(); err != nil {
				return err
			}
		}
	}
	return nil
}
