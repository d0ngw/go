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
		D []int ",flow"
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
    desc: "字符串不恩能够为空,且最小长度为1,最大长度为2"
    validators:
    - min: "1"
      name: strlen
      max: "2"
    - name: notempty
validates2:
  sname: v2
  rules:
  - name: minStr
    desc: "字符串不恩能够为空,且最小长度为1,最大长度为5"
    validators:
    - min: "1"
      name: strlen
      max: "5"
    - name: notempty
  - name: allowempty
    desc: "字符串可以为空,如果不为空,则最小长度为2,最大长度为5"
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
