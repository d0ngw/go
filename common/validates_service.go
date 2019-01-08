package common

import (
	"fmt"
	"strings"
)

// RuleConfig 验证规则配置
type RuleConfig struct {
	Name       string
	Desc       string              //规则描述
	Validators []map[string]string //验证器列表,必须要有name
}

// ValidateRuleConfig 验证规则配置
type ValidateRuleConfig struct {
	Rules  []RuleConfig
	SName  string          //服务的名称
	parsed validateRuleMap //解析后的结果
}

// Parse 解析验证的配置
func (p *ValidateRuleConfig) Parse() error {
	if p == nil {
		Warnf("no validate conf")
		return nil
	}
	rules := make(validateRuleMap)
	for _, ruleConfig := range p.Rules {
		ruleName := strings.TrimSpace(ruleConfig.Name)
		if len(ruleName) == 0 {
			panic("The rule name must not be empty")
		}
		validators := make([]StrValidator, 0, len(ruleConfig.Validators))
		for _, validatorConf := range ruleConfig.Validators {
			validators = append(validators, NewValidatorByConf(validatorConf))
		}
		rule := &ValidateRule{
			desc:       ruleConfig.Desc,
			validators: validators}
		rules[ruleName] = rule
		Debugf("Add validate rule:%s", ruleName)
	}
	p.parsed = rules
	return nil
}

// NewService 根据配置解析的结果创建验证服务
func (p *ValidateRuleConfig) NewService() Service {
	if p.parsed == nil {
		panic("Can't create ValidateService from nil")
	}
	svr := RuleValidateService{}
	svr.SName = p.SName
	svr.rules = p.parsed
	return &svr
}

//ValidateRule 定义验证规则
type ValidateRule struct {
	desc       string         //规则描述
	validators []StrValidator //通过Rules构建出来的验证规则
}

type validateRuleMap map[string]*ValidateRule

// ValidateConfigurer validateConfig
type ValidateConfigurer interface {
	GetValidateRuleConfig() *ValidateRuleConfig
}

// ValidateService 验证服务
type ValidateService interface {
	Service
	//Validate 使用name指定验证规则,对value进行验证,验证通过返回nil,否则返回错误原因
	Validate(name string, value string) error
}

// RuleValidateService  根据规则进行的验证服务
type RuleValidateService struct {
	BaseService
	Config ValidateConfigurer `inject:"_"`
	rules  validateRuleMap
}

// Init implements Initable
func (p *RuleValidateService) Init() error {
	if p.Config == nil || p.Config.GetValidateRuleConfig() == nil {
		return fmt.Errorf("no validate config")
	}
	config := p.Config.GetValidateRuleConfig()
	p.SName = config.SName
	p.rules = config.parsed
	return nil
}

// Validate 验证
func (p *RuleValidateService) Validate(ruleName string, s string) error {
	rule := p.rules[ruleName]
	if rule == nil {
		return fmt.Errorf("can't find validate rule %s", ruleName)
	}

	for _, v := range rule.validators {
		if !v.Validate(s) {
			return NewValidateError(rule.desc)
		}
	}
	return nil
}

// ValidatePair 定义验证规则名称其需要验证的值
type ValidatePair struct {
	Name  string
	Value string
	Msg   string
}

// NewValidatePair create ValidatePair
func NewValidatePair(name, value string) *ValidatePair {
	return &ValidatePair{Name: name, Value: value}
}

// NewValidatePairMsg create ValidatePair with msg
func NewValidatePairMsg(name, value, msg string) *ValidatePair {
	return &ValidatePair{Name: name, Value: value, Msg: msg}
}

// ValidateAll 验证所有的规则
func ValidateAll(validateService ValidateService, nameAndValues ...*ValidatePair) error {
	for _, nv := range nameAndValues {
		if err := validateService.Validate(nv.Name, nv.Value); err != nil {
			if nv.Msg != "" {
				return NewValidateError(nv.Msg)
			}
			return err
		}
	}
	return nil
}

// ValidateError error
type ValidateError struct {
	msg string //错误消息
}

// NewValidateError new
func NewValidateError(msg string) *ValidateError {
	return &ValidateError{msg: msg}
}

func (p *ValidateError) Error() string {
	if p == nil {
		return ""
	}
	return p.msg
}

// Human impls HumanError.Human
func (p *ValidateError) Human() bool {
	if p == nil {
		return false
	}
	return true
}
