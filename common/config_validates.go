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
			return fmt.Errorf("%s", rule.desc)
		}
	}
	return nil
}
