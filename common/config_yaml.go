package common

import (
	"errors"
	"fmt"
	"os"
	"path"

	yaml "gopkg.in/yaml.v3"
)

// LoadYAMLFromPath loads a YAML file into target.
func LoadYAMLFromPath(filename string, target interface{}) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return LoadYAMl(data, target)
}

// LoadYAMl decodes YAML data into target.
func LoadYAMl(data []byte, target interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("can't load yaml config from empty data")
	}
	return yaml.Unmarshal([]byte(data), target)
}

// LoadConfig loads and combines the YAML files under configDir.
func LoadConfig(config Configurer, addonConfig string, configDir string, pathes ...string) (err error) {
	return LoadConfigWithLoader(FileLoader, config, addonConfig, configDir, pathes...)
}

// LoadConfigWithLoader loads configuration using the supplied loader.
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
