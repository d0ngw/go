package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ValidatePair 定义验证规则名称其需要验证的值
type ValidatePair struct {
	Name  string
	Value string
}

// ValidateService 验证服务
type ValidateService interface {
	Service
	//Validate 使用name指定验证规则,对value进行验证,验证通过返回nil,否则返回错误原因
	Validate(name string, value string) error
}

// ValidateAll 验证所有的规则
func ValidateAll(validateService ValidateService, nameAndValues ...*ValidatePair) error {
	for _, nv := range nameAndValues {
		if err := validateService.Validate(nv.Name, nv.Value); err != nil {
			return err
		}
	}
	return nil
}

// StrValidator 字符串验证器
type StrValidator interface {
	//Vlidate 验证字符串参数是否符合规则
	Validate(param string) bool
}

// StringLenValidator 字符串长度验证
type StringLenValidator struct {
	min int //最小长度
	max int //最大长度
}

// Validate 验证字符串的长度
func (p *StringLenValidator) Validate(param string) bool {
	strLen := len(param)
	return p.min <= strLen && strLen <= p.max
}

// NotEmptyValidator 非空
type NotEmptyValidator struct {
}

// Validate 验证字符串是否为空
func (p *NotEmptyValidator) Validate(param string) bool {
	if len(param) == 0 {
		return false
	}
	return len(strings.TrimSpace(param)) > 0
}

// Int32Validator 32位整数验证
type Int32Validator struct {
	min int32 //最小值
	max int32 //最大值
}

// Validate 验证整型值
func (p *Int32Validator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if v, err := strconv.ParseInt(param, 10, 32); err == nil {
		vi32 := int32(v)
		return p.min <= vi32 && vi32 <= p.max
	}
	return false
}

// Int64Validator 64位整数验证
type Int64Validator struct {
	min int64 //最小值
	max int64 //最大值
}

// Validate 验证整型值
func (p *Int64Validator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if v, err := strconv.ParseInt(param, 10, 64); err == nil {
		return p.min <= v && v <= p.max
	}
	return false
}

// Float32Validator  32位浮点数验证
type Float32Validator struct {
	min float32 //最小值
	max float32 //最大值
}

// Validate 验证浮点数值
func (p *Float32Validator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if v, err := strconv.ParseFloat(param, 32); err == nil {
		v32 := float32(v)
		return p.min <= v32 && v32 <= p.max
	}
	return false
}

// Float64Validator  64位浮点数验证
type Float64Validator struct {
	min float64 //最小值
	max float64 //最大值
}

// Validate 验证浮点数值
func (p *Float64Validator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if v, err := strconv.ParseFloat(param, 64); err == nil {
		return p.min <= v && v <= p.max
	}
	return false
}

// BoolValidator  bool验证
type BoolValidator struct {
}

// Validate 验证bool值
func (p *BoolValidator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if _, err := strconv.ParseBool(param); err == nil {
		return true
	}
	return false
}

// RegExValidator 正则表达式验证
type RegExValidator struct {
	pattern *regexp.Regexp //正则表达式
	empty   bool           //是否允许为空
}

// Validate 正则表达式验证
func (p *RegExValidator) Validate(param string) bool {
	if param == "" && p.empty {
		return true
	}
	return p.pattern.MatchString(param)
}

// ParseInt 解析整数
func ParseInt(param string) (v int, err error) {
	v, err = strconv.Atoi(param)
	return
}

// ParseInt32 解析整数
func ParseInt32(param string) (v int32, err error) {
	v64, err := strconv.ParseInt(param, 10, 32)
	if err == nil {
		v = int32(v64)
	}
	return
}

// ParseInt64 解析整数
func ParseInt64(param string) (v int64, err error) {
	v, err = strconv.ParseInt(param, 10, 64)
	return
}

// ParseFloat32 解析浮点数
func ParseFloat32(param string) (v float32, err error) {
	v64, err := strconv.ParseFloat(param, 32)
	if err == nil {
		v = float32(v64)
	}
	return
}

// ParseFloat64 解析浮点数
func ParseFloat64(param string) (v float64, err error) {
	v, err = strconv.ParseFloat(param, 64)
	return
}

// ValidatorNewer 创建验证器的函数类型
type ValidatorNewer func(conf map[string]string) StrValidator

// NewNotEmptyValidator 创建非空验证器
func NewNotEmptyValidator(conf map[string]string) StrValidator {
	return vNOTEMPTY
}

// NewBoolValidator 创建bool验证器
func NewBoolValidator(conf map[string]string) StrValidator {
	return vBOOL
}

// NewStrLenValidator 创建字符串长度验证,conf["min"],最小值;conf["max"],最大值
func NewStrLenValidator(conf map[string]string) StrValidator {
	minLen, err := ParseInt(conf["min"])
	if err != nil {
		panic(err)
	}
	maxLen, err := ParseInt(conf["max"])
	if err != nil {
		panic(err)
	}
	if minLen < 0 || maxLen < 0 || minLen > maxLen {
		panic(fmt.Errorf("Invalid str length,minLen:%v,maxLen:%v", minLen, maxLen))
	}
	return &StringLenValidator{min: minLen, max: maxLen}
}

// NewInt32Validator 创建int32验证,conf["min"],最小值;conf["max"],最大值
func NewInt32Validator(conf map[string]string) StrValidator {
	min, err := ParseInt32(conf["min"])
	if err != nil {
		panic(err)
	}
	max, err := ParseInt32(conf["max"])
	if err != nil {
		panic(err)
	}
	if min > max {
		panic(fmt.Errorf("Invalid min %d,max %d", min, max))
	}
	return &Int32Validator{min: min, max: max}

}

// NewInt64Validator 创建int64验证,conf["min"],最小值;conf["max"],最大值
func NewInt64Validator(conf map[string]string) StrValidator {
	min, err := ParseInt64(conf["min"])
	if err != nil {
		panic(err)
	}
	max, err := ParseInt64(conf["max"])
	if err != nil {
		panic(err)
	}
	if min > max {
		panic(fmt.Errorf("Invalid min %d,max %d", min, max))
	}
	return &Int64Validator{min: min, max: max}
}

// NewFloat32Validator  创建float32验证,conf["min"],最小值;conf["max"],最大值
func NewFloat32Validator(conf map[string]string) StrValidator {
	min, err := ParseFloat32(conf["min"])
	if err != nil {
		panic(err)
	}
	max, err := ParseFloat32(conf["max"])
	if err != nil {
		panic(err)
	}
	if min > max {
		panic(fmt.Errorf("Invalid min %f,max %f", min, max))
	}
	return &Float32Validator{min: min, max: max}
}

// NewFloat64Validator 创建float64验证,conf["min"],最小值;conf["max"],最大值
func NewFloat64Validator(conf map[string]string) StrValidator {
	min, err := ParseFloat64(conf["min"])
	if err != nil {
		panic(err)
	}
	max, err := ParseFloat64(conf["max"])
	if err != nil {
		panic(err)
	}
	if min > max {
		panic(fmt.Errorf("Invalid min %f,max %f", min, max))
	}
	return &Float64Validator{min: min, max: max}
}

// NewRegexValidator 创建正则表达式验证,conf["pattern"] 正则表达式
func NewRegexValidator(conf map[string]string) StrValidator {
	pattern := conf["pattern"]
	allowEmpty := "true" == strings.ToLower(conf["empty"])
	if len(pattern) == 0 {
		panic(fmt.Errorf("Invalid pattern %s", pattern))
	}
	return &RegExValidator{pattern: regexp.MustCompile(pattern), empty: allowEmpty}
}

//默认的构建器的名称
const (
	VNOTEMPTY = "notempty" //无构建参数
	VBOOL     = "bool"
	VSTRLEN   = "strlen"
	VINT32    = "i32"
	VINT64    = "i64"
	VFLOAT32  = "f32"
	VFLOAT64  = "f64"
	VREGEX    = "regex"
)

var (
	vNOTEMPTY        = &NotEmptyValidator{}
	vBOOL            = &BoolValidator{}
	validateRegister = NewCopyOnWriteMap()
)

// RegValidatorNewer 根据名称注册验证器构建函数
func RegValidatorNewer(name string, validator ValidatorNewer) {
	if err := validateRegister.PutIfAbsent(name, validator); err != nil {
		panic("Duplicate validator " + err.Error())
	}
}

// NewValidatorByConf 根据配置conf["name"]及其对应的参数构建验证器
func NewValidatorByConf(conf map[string]string) StrValidator {
	name := conf["name"]
	if f := validateRegister.Get(name); f != nil {
		return f.(ValidatorNewer)(conf)
	}
	panic("Can't find the validator name:" + name)
}

//初始化注册内置的验证器
func init() {
	RegValidatorNewer(VNOTEMPTY, NewNotEmptyValidator)
	RegValidatorNewer(VBOOL, NewBoolValidator)
	RegValidatorNewer(VSTRLEN, NewStrLenValidator)
	RegValidatorNewer(VINT32, NewInt32Validator)
	RegValidatorNewer(VINT64, NewInt64Validator)
	RegValidatorNewer(VFLOAT32, NewFloat32Validator)
	RegValidatorNewer(VFLOAT64, NewFloat64Validator)
	RegValidatorNewer(VREGEX, NewRegexValidator)
}
