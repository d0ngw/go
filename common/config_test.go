package common

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

var data = `
a: Easy!
b:
  c: 2
  d: [3, 4]
`

type conf struct {
	A string
	B struct {
		C int
		D []int
	}
}

func TestLoadYAML(t *testing.T) {
	config := conf{}
	LoadYAMl([]byte(data), &config)
	assert.Equal(t, "Easy!", config.A)
	assert.Equal(t, 2, config.B.C)
	assert.Equal(t, 2, len(config.B.D))
	assert.Equal(t, []int{3, 4}, config.B.D)
}

var appConfigData = `validates:
  sname: v1
  rules:
  - name: minStr
    desc: "The string must not be empty and must contain 1 to 2 characters."
    validators:
    - min: "1"
      name: strlen
      max: "2"
    - name: notempty
validates2:
  sname: v2
  rules:
  - name: minStr
    desc: "The string must not be empty and must contain 1 to 5 characters."
    validators:
    - min: "1"
      name: strlen
      max: "5"
    - name: notempty
  - name: allowempty
    desc: "The string may be empty; non-empty values must contain at most 5 characters."
    validators:
    - min: "0"
      name: strlen
      max: "5"
`

type ConfigTest struct {
	AppConfig `yaml:",inline"`
	V2        *ValidateRuleConfig `yaml:"validates2"`
}

func TestAppConfig(t *testing.T) {
	var appConfig ConfigTest
	LoadYAMl([]byte(appConfigData), &appConfig)
	Parse(&appConfig)
	fmt.Println("validates:", appConfig.ValidateRuleConfig.parsed)
	fmt.Println("validates:", appConfig.V2.parsed)
	v2 := appConfig.V2.NewService().(ValidateService)
	v1 := appConfig.ValidateRuleConfig.NewService().(ValidateService)
	fmt.Printf("v1 name:%s,v2 name:%s\n", v1.Name(), v2.Name())
	err := v1.Validate("minStr", "")
	assert.NotNil(t, err)
	err = v2.Validate("allowempty", "")
	assert.Nil(t, err)

	err = v1.Validate("minStr", "he")
	assert.Nil(t, err)

	err = v2.Validate("minStr", "hello")
	assert.Nil(t, err)

	var s1 ValidateService
	var s2 interface{}

	fmt.Printf("s1 type:%T,size:%d\n", s1, unsafe.Sizeof(s1))
	fmt.Printf("s2 type:%T,size:%d\n", s2, unsafe.Sizeof(s2))
}

func TestConfigLoader(t *testing.T) {
	fileLoader := &ConfigFileLoader{}
	exist, err := fileLoader.Exist("")
	assert.False(t, exist)
	assert.NoError(t, err)

	exist, err = fileLoader.Exist("config_test.go")
	assert.True(t, exist)
	assert.NoError(t, err)
}
