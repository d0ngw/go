package common

import (
	"fmt"
	"strings"
)

// RuleConfig configures a validation rule.
type RuleConfig struct {
	Name       string
	Desc       string              // Rule description.
	Validators []map[string]string // Validator configurations, each requiring a name.
}

// ValidateRuleConfig configures named validation rules.
type ValidateRuleConfig struct {
	Rules  []RuleConfig
	SName  string          // Service name.
	parsed validateRuleMap // Parsed rules.
}

// Parse parses and validates the validation rules.
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
	}
	p.parsed = rules
	return nil
}

// NewService creates a validation service from the parsed configuration.
func (p *ValidateRuleConfig) NewService() Service {
	if p.parsed == nil {
		panic("Can't create ValidateService from nil")
	}
	svr := RuleValidateService{}
	svr.SName = p.SName
	svr.rules = p.parsed
	return &svr
}

// ValidateRule contains a parsed validation rule.
type ValidateRule struct {
	desc       string         // Rule description.
	validators []StrValidator // Validators built from the rule configuration.
}

type validateRuleMap map[string]*ValidateRule

// ValidateConfigurer validateConfig
type ValidateConfigurer interface {
	GetValidateRuleConfig() *ValidateRuleConfig
}

// ValidateService validates values using named rules.
type ValidateService interface {
	Service
	// Validate checks value against the named rule and returns an error on failure.
	Validate(name string, value string) error
}

// RuleValidateService validates values using named rules.
type RuleValidateService struct {
	BaseService
	Config ValidateConfigurer `inject:"_"`
	rules  validateRuleMap
}

// Init implements Initable
func (p *RuleValidateService) Init() error {
	if p.Config == nil {
		Warnf("no validate config")
		return nil
	}
	if p.Config.GetValidateRuleConfig() == nil {
		Warnf("no validate config rule")
		return nil
	}
	config := p.Config.GetValidateRuleConfig()
	p.SName = config.SName
	p.rules = config.parsed
	return nil
}

// Validate checks a value against the rule.
func (p *RuleValidateService) Validate(ruleName string, s string) error {
	rule := p.rules[ruleName]
	if rule == nil {
		err := NewValidateError(fmt.Sprintf("can't find validate rule %s", ruleName))
		err.notFoundRule = true
		return err
	}

	for _, v := range rule.validators {
		if !v.Validate(s) {
			return NewValidateError(rule.desc)
		}
	}
	return nil
}

// ValidatePair associates a rule name with a value to validate.
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

// ValidateAll validates all supplied rule and value pairs.
func ValidateAll(validateService ValidateService, nameAndValues ...*ValidatePair) error {
	for _, nv := range nameAndValues {
		if err := validateService.Validate(nv.Name, nv.Value); err != nil {
			if verr, ok := err.(*ValidateError); ok && verr.notFoundRule {
				return err
			}
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
	// Error message.
	msg string
	// Whether the rule was not found.
	notFoundRule bool
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
