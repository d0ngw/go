package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ValidatePair struct {
	Name  string
	Value string
}

//验证服务
type ValidateService interface {
	Service
	//Validate 使用name指定验证规则,对value进行验证,验证通过返回nil,否则返回错误原因
	Validate(name string, value string) error
	//ValidateAll 验证所有的规则,验证通过返回nil,否则返回错误原因
	ValidateAll(nameAndValues ...*ValidatePair) error
}

type BaseValidateService struct {
	BaseService
}

func (p *BaseValidateService) Validate(name, value string) error {
	panic("Please implement Validate function")
}

func (p *BaseValidateService) ValidateAll(nameAndValues ...*ValidatePair) error {
	for _, nv := range nameAndValues {
		if err := p.Validate(nv.Name, nv.Value); err != nil {
			return err
		}
	}
	return nil
}

//字符串验证器
type StrValidator interface {
	//Vlidate 验证字符串参数是否符合规则
	Validate(param string) bool
}

//字符串长度验证
type StringLenValidator struct {
	min int //最小长度
	max int //最大长度
}

func (self *StringLenValidator) Validate(param string) bool {
	strLen := len(param)
	return self.min <= strLen && strLen <= self.max
}

//非空
type NotEmptyValidator struct {
}

func (self *NotEmptyValidator) Validate(param string) bool {
	if len(param) == 0 {
		return false
	}
	return len(strings.TrimSpace(param)) > 0
}

//32位整数验证
type Int32Validator struct {
	min int32 //最小值
	max int32 //最大值
}

func (self *Int32Validator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if v, err := strconv.ParseInt(param, 10, 32); err == nil {
		vi32 := int32(v)
		return self.min <= vi32 && vi32 <= self.max
	} else {
		return false
	}
}

//64位整数验证
type Int64Validator struct {
	min int64 //最小值
	max int64 //最大值
}

func (self *Int64Validator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if v, err := strconv.ParseInt(param, 10, 64); err == nil {
		return self.min <= v && v <= self.max
	} else {
		return false
	}
}

//32位浮点数验证
type Float32Validator struct {
	min float32 //最小值
	max float32 //最大值
}

func (self *Float32Validator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if v, err := strconv.ParseFloat(param, 32); err == nil {
		v32 := float32(v)
		return self.min <= v32 && v32 <= self.max
	} else {
		return false
	}
}

//64位浮点数验证
type Float64Validator struct {
	min float64 //最小值
	max float64 //最大值
}

func (self *Float64Validator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if v, err := strconv.ParseFloat(param, 64); err == nil {
		return self.min <= v && v <= self.max
	} else {
		return false
	}
}

//Bool验证
type BoolValidator struct {
}

func (self *BoolValidator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if _, err := strconv.ParseBool(param); err == nil {
		return true
	} else {
		return false
	}
}

//正则表达式验证
type RegExValidator struct {
	pattern *regexp.Regexp //正则表达式
}

func (self *RegExValidator) Validate(param string) bool {
	return self.pattern.MatchString(param)
}

func ParseInt(param string) int {
	if v, err := strconv.Atoi(param); err == nil {
		return v
	} else {
		panic(err)
	}
}

func ParseInt32(param string) int32 {
	if v, err := strconv.ParseInt(param, 10, 32); err == nil {
		return int32(v)
	} else {
		panic(err)
	}
}

func ParseInt64(param string) int64 {
	if v, err := strconv.ParseInt(param, 10, 64); err == nil {
		return int64(v)
	} else {
		panic(err)
	}
}

func ParseFloat32(param string) float32 {
	if v, err := strconv.ParseFloat(param, 32); err == nil {
		return float32(v)
	} else {
		panic(err)
	}
}

func ParseFloat64(param string) float64 {
	if v, err := strconv.ParseFloat(param, 64); err == nil {
		return float64(v)
	} else {
		panic(err)
	}
}

type ValidatorNewer func(conf map[string]string) StrValidator

func NewNotEmptyValidator(conf map[string]string) StrValidator {
	return _V_NOT_EMPTY
}

func NewBoolValidator(conf map[string]string) StrValidator {
	return _V_BOOL
}

//创建字符串长度验证,conf["min"],最小值;conf["max"],最大值
func NewStrLenValidator(conf map[string]string) StrValidator {
	minLen := ParseInt(conf["min"])
	maxLen := ParseInt(conf["max"])
	if minLen < 0 || maxLen < 0 || minLen > maxLen {
		panic(fmt.Errorf("Invalid str length,minLen:%v,maxLen:%v", minLen, maxLen))
	}
	return &StringLenValidator{
		min: minLen,
		max: maxLen}
}

//创建int32验证,conf["min"],最小值;conf["max"],最大值
func NewInt32Validator(conf map[string]string) StrValidator {
	min := ParseInt32(conf["min"])
	max := ParseInt32(conf["max"])
	if min > max {
		panic(fmt.Errorf("Invalid min %d,max %d", min, max))
	}
	return &Int32Validator{
		min: min,
		max: max}
}

//创建int64验证,conf["min"],最小值;conf["max"],最大值
func NewInt64Validator(conf map[string]string) StrValidator {
	min := ParseInt64(conf["min"])
	max := ParseInt64(conf["max"])
	if min > max {
		panic(fmt.Errorf("Invalid min %d,max %d", min, max))
	}
	return &Int64Validator{
		min: min,
		max: max}
}

//创建float32验证,conf["min"],最小值;conf["max"],最大值
func NewFloat32Validator(conf map[string]string) StrValidator {
	min := ParseFloat32(conf["min"])
	max := ParseFloat32(conf["max"])
	if min > max {
		panic(fmt.Errorf("Invalid min %d,max %d", min, max))
	}
	return &Float32Validator{
		min: min,
		max: max}
}

//创建float64验证,conf["min"],最小值;conf["max"],最大值
func NewFloat64Validator(conf map[string]string) StrValidator {
	min := ParseFloat64(conf["min"])
	max := ParseFloat64(conf["max"])
	if min > max {
		panic(fmt.Errorf("Invalid min %d,max %d", min, max))
	}
	return &Float64Validator{
		min: min,
		max: max}
}

//创建正则表达式验证,conf["pattern"] 正则表达式
func NewRegexValidator(conf map[string]string) StrValidator {
	pattern := conf["pattern"]
	if len(pattern) == 0 {
		panic(fmt.Errorf("Invalid pattern %s", pattern))
	}
	return &RegExValidator{regexp.MustCompile(pattern)}
}

//默认的构建器的名称
const (
	V_NOT_EMPTY = "notempty" //无构建参数
	V_BOOL      = "bool"
	V_STR_LEN   = "strlen"
	V_INT32     = "i32"
	V_INT64     = "i64"
	V_FLOAT32   = "f32"
	V_FLOAT64   = "f64"
	V_REGEX     = "regex"
)

var (
	_V_NOT_EMPTY = &NotEmptyValidator{}
	_V_BOOL      = &BoolValidator{}
	//按名字索引的验证构建器
	validateRegister = NewCopyOnWriteMap()
)

//根据名称注册验证器构建函数
func RegValidatorNewer(name string, validator ValidatorNewer) {
	if err := validateRegister.PutIfAbsent(name, validator); err != nil {
		panic("Duplicate validator " + err.Error())
	}
}

//根据配置conf["name"]及其对应的参数构建验证器
func NewValidatorByConf(conf map[string]string) StrValidator {
	name := conf["name"]
	if f := validateRegister.Get(name); f != nil {
		return f.(ValidatorNewer)(conf)
	} else {
		panic("Can't find the validator name:" + name)
	}
}

//初始化注册内置的验证器
func init() {
	RegValidatorNewer(V_NOT_EMPTY, NewNotEmptyValidator)
	RegValidatorNewer(V_BOOL, NewBoolValidator)
	RegValidatorNewer(V_STR_LEN, NewStrLenValidator)
	RegValidatorNewer(V_INT32, NewInt32Validator)
	RegValidatorNewer(V_INT64, NewInt64Validator)
	RegValidatorNewer(V_FLOAT32, NewFloat32Validator)
	RegValidatorNewer(V_FLOAT64, NewFloat64Validator)
	RegValidatorNewer(V_REGEX, NewRegexValidator)
}
