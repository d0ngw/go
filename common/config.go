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

// ConfigLoader loads configuration content.
type ConfigLoader interface {
	Load(configPath string) (content []byte, err error)

	Exist(configPath string) (exist bool, err error)
}

// ConfigFileLoader loads configuration from local files.
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
	// FileLoader is the default configuration loader.
	FileLoader ConfigLoader = &ConfigFileLoader{}
)

// Configurer parses and validates configuration.
type Configurer interface {
	// Parse parses the configuration.
	Parse() error
}

// LogConfig configures logging.
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

// Parse configures logging.
func (p *LogConfig) Parse() error {
	return initLogger(p)
}

// GetOutputFile returns the log output file name.
func (p *LogConfig) GetOutputFile() string {
	if p == nil {
		return ""
	}

	name := os.Getenv("LOG_FILE")
	if name != "" {
		return name
	}
	return p.FileName
}

// RuntimeConfig configures runtime behavior.
type RuntimeConfig struct {
	Maxprocs int // Maximum number of OS threads executing Go code simultaneously.
}

// Parse applies the runtime settings.
func (p *RuntimeConfig) Parse() error {
	if p.Maxprocs > 0 {
		preProcs := runtime.GOMAXPROCS(p.Maxprocs)
		Infof("Set runtime.MAXPROCS to %v,old is %v", p.Maxprocs, preProcs)
	}
	return nil
}

// AppConfig contains common application settings.
type AppConfig struct {
	*LogConfig          `yaml:"log"`
	*RuntimeConfig      `yaml:"runtime"`
	*ValidateRuleConfig `yaml:"validates"`
}

// Parse parses the common application configuration.
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

// Parse parses the configuration.
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
