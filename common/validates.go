package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// StrValidator validates string values.
type StrValidator interface {
	// Validate reports whether the string satisfies the rule.
	Validate(param string) bool
}

// StringLenValidator validates string length.
type StringLenValidator struct {
	min int // Minimum length.
	max int // Maximum length.
}

// Validate checks the string length.
func (p *StringLenValidator) Validate(param string) bool {
	strLen := utf8.RuneCountInString(param)
	return p.min <= strLen && strLen <= p.max
}

// NotEmptyValidator rejects empty strings.
type NotEmptyValidator struct {
}

// Validate checks that the string is not empty.
func (p *NotEmptyValidator) Validate(param string) bool {
	if len(param) == 0 {
		return false
	}
	return len(strings.TrimSpace(param)) > 0
}

// Int32Validator validates 32-bit integers.
type Int32Validator struct {
	min int32 // Minimum value.
	max int32 // Maximum value.
}

// Validate checks an integer value.
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

// Int64Validator validates 64-bit integers.
type Int64Validator struct {
	min int64 // Minimum value.
	max int64 // Maximum value.
}

// Validate checks an integer value.
func (p *Int64Validator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if v, err := strconv.ParseInt(param, 10, 64); err == nil {
		return p.min <= v && v <= p.max
	}
	return false
}

// Float32Validator validates 32-bit floating-point values.
type Float32Validator struct {
	min float32 // Minimum value.
	max float32 // Maximum value.
}

// Validate checks a floating-point value.
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

// Float64Validator validates 64-bit floating-point values.
type Float64Validator struct {
	min float64 // Minimum value.
	max float64 // Maximum value.
}

// Validate checks a floating-point value.
func (p *Float64Validator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if v, err := strconv.ParseFloat(param, 64); err == nil {
		return p.min <= v && v <= p.max
	}
	return false
}

// BoolValidator validates boolean values.
type BoolValidator struct {
}

// Validate checks a boolean value.
func (p *BoolValidator) Validate(param string) bool {
	if len(param) == 0 {
		return true
	}
	if _, err := strconv.ParseBool(param); err == nil {
		return true
	}
	return false
}

// RegExValidator validates strings using a regular expression.
type RegExValidator struct {
	pattern *regexp.Regexp // Regular expression.
	empty   bool           // Whether empty values are allowed.
}

// Validate checks whether a value matches the regular expression.
func (p *RegExValidator) Validate(param string) bool {
	if param == "" && p.empty {
		return true
	}
	return p.pattern.MatchString(param)
}

// ParseInt parses an integer.
func ParseInt(param string) (v int, err error) {
	v, err = strconv.Atoi(param)
	return
}

// ParseInt32 parses a 32-bit integer.
func ParseInt32(param string) (v int32, err error) {
	v64, err := strconv.ParseInt(param, 10, 32)
	if err == nil {
		v = int32(v64)
	}
	return
}

// ParseInt64 parses a 64-bit integer.
func ParseInt64(param string) (v int64, err error) {
	v, err = strconv.ParseInt(param, 10, 64)
	return
}

// ParseFloat32 parses a 32-bit floating-point value.
func ParseFloat32(param string) (v float32, err error) {
	v64, err := strconv.ParseFloat(param, 32)
	if err == nil {
		v = float32(v64)
	}
	return
}

// ParseFloat64 parses a 64-bit floating-point value.
func ParseFloat64(param string) (v float64, err error) {
	v, err = strconv.ParseFloat(param, 64)
	return
}

// ValidatorNewer constructs a validator from configuration.
type ValidatorNewer func(conf map[string]string) StrValidator

// NewNotEmptyValidator creates a validator that rejects empty strings.
func NewNotEmptyValidator(conf map[string]string) StrValidator {
	return vNOTEMPTY
}

// NewBoolValidator creates a boolean validator.
func NewBoolValidator(conf map[string]string) StrValidator {
	return vBOOL
}

// NewStrLenValidator creates a string length validator using conf["min"] and conf["max"].
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
		panic(fmt.Errorf("invalid str length,minLen:%v,maxLen:%v", minLen, maxLen))
	}
	return &StringLenValidator{min: minLen, max: maxLen}
}

// NewInt32Validator creates an int32 validator using conf["min"] and conf["max"].
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
		panic(fmt.Errorf("invalid min %d,max %d", min, max))
	}
	return &Int32Validator{min: min, max: max}

}

// NewInt64Validator creates an int64 validator using conf["min"] and conf["max"].
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
		panic(fmt.Errorf("invalid min %d,max %d", min, max))
	}
	return &Int64Validator{min: min, max: max}
}

// NewFloat32Validator creates a float32 validator using conf["min"] and conf["max"].
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
		panic(fmt.Errorf("invalid min %f,max %f", min, max))
	}
	return &Float32Validator{min: min, max: max}
}

// NewFloat64Validator creates a float64 validator using conf["min"] and conf["max"].
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
		panic(fmt.Errorf("invalid min %f,max %f", min, max))
	}
	return &Float64Validator{min: min, max: max}
}

// NewRegexValidator creates a validator using the regular expression in conf["pattern"].
func NewRegexValidator(conf map[string]string) StrValidator {
	pattern := conf["pattern"]
	allowEmpty := strings.ToLower(conf["empty"]) == "true"
	if len(pattern) == 0 {
		panic(fmt.Errorf("invalid pattern %s", pattern))
	}
	return &RegExValidator{pattern: regexp.MustCompile(pattern), empty: allowEmpty}
}

// Built-in validator constructor names.
const (
	VNOTEMPTY = "notempty" // No constructor parameters.
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

// RegValidatorNewer registers a validator constructor by name.
func RegValidatorNewer(name string, validator ValidatorNewer) {
	if err := validateRegister.PutIfAbsent(name, validator); err != nil {
		panic("Duplicate validator " + err.Error())
	}
}

// NewValidatorByConf creates a validator using conf["name"] and its parameters.
func NewValidatorByConf(conf map[string]string) StrValidator {
	name := conf["name"]
	if f := validateRegister.Get(name); f != nil {
		return f.(ValidatorNewer)(conf)
	}
	panic("Can't find the validator name:" + name)
}

// Register built-in validators.
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
