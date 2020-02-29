package common

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"

	yaml "gopkg.in/yaml.v2"
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
		return fmt.Errorf("Can't load yaml config from empty data")
	}
	return yaml.Unmarshal([]byte(data), target)
}

// LoadConfig 从configDir目录下的多个path指定的YAML配置文件中加载配置
func LoadConfig(config Configurer, addonConfig string, configDir string, pathes ...string) (err error) {
	return LoadConfigWithLoader(FileLoader, config, addonConfig, configDir, pathes...)
}

// LoadConfigWithLoader 使用指定的加载器加载配置
func LoadConfigWithLoader(loader ConfigLoader, config Configurer, addonConfig string, configDir string, pathes ...string) (err error) {
	if loader == nil {
		err = errors.New("no loader")
		return
	}
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
		cnt, err := loader.Load(p)
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
