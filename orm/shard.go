package orm

import (
	"fmt"
	"sort"
	"strconv"

	c "github.com/d0ngw/go/common"
)

// ShardPolicy 分片规则
type ShardPolicy string

// ShardRule 分片规则的实现
type ShardRule interface {
	c.Configurer
	// Policy 返回策略名称
	Policy() ShardPolicy
	// Shard 计算分片的名称
	Shard(val interface{}) (shardName string, err error)
	// ShardFieldName 用于分片的字段名称
	ShardFieldName() string
}

const (
	//Hash hash shard
	Hash ShardPolicy = "hash"
	//Named named shard
	Named ShardPolicy = "named"
	//NumRange number range shard
	NumRange ShardPolicy = "num_range"
)

// IsValid 是否有效
func (p ShardPolicy) IsValid() bool {
	return p == Hash || p == Named || p == NumRange
}

// HashRule hash规则
type HashRule struct {
	Count      int64  `yaml:"count"`       //hash的个数
	NamePrefix string `yaml:"name_prefix"` //名称的前缀
	FieldName  string `yaml:"field_name"`  //hash取值的字段名
}

// Policy implements ShardRule
func (p *HashRule) Policy() ShardPolicy {
	return Hash
}

// Parse implements Configurer
func (p *HashRule) Parse() error {
	if p.Count <= 0 {
		return fmt.Errorf("invalid count")
	}
	if p.NamePrefix == "" {
		return fmt.Errorf("invalid name_prefix")
	}
	if p.FieldName == "" {
		return fmt.Errorf("invalid field_name")
	}
	return nil
}

// Shard implements ShardRule.Shard
func (p *HashRule) Shard(val interface{}) (shardName string, err error) {
	valInt64, err := c.Int64(val)
	if err != nil {
		return
	}
	if valInt64 < 0 {
		return "", fmt.Errorf("invalid hash val %v", val)
	}
	return p.NamePrefix + strconv.FormatInt((valInt64%p.Count), 10), nil
}

// ShardFieldName 用于分片的字段名
func (p *HashRule) ShardFieldName() string {
	return p.FieldName
}

// NamedRule 指定命名
type NamedRule struct {
	Name string `yaml:"name"`
}

// Policy implements ShardRule
func (p *NamedRule) Policy() ShardPolicy {
	return Named
}

// Parse implements Configurer
func (p *NamedRule) Parse() error {
	if p.Name == "" {
		return fmt.Errorf("invalid name")
	}
	return nil
}

// Shard implements ShardRule.Shard
func (p *NamedRule) Shard(val interface{}) (shardName string, err error) {
	return p.Name, nil
}

// ShardFieldName 用于分片的字段名
func (p *NamedRule) ShardFieldName() string {
	return ""
}

// NumRangeRule 数字区间
type NumRangeRule struct {
	FieldName   string `yaml:"field_name"` //分片取值的字段名
	DefaultName string `yaml:"default_name"`
	Ranges      []*struct {
		Begin int64  `yaml:"begin"`
		End   int64  `yaml:"end"`
		Name  string `yaml:"name"`
	} `yaml:"ranges"`
}

// Policy implements ShardRule
func (p *NumRangeRule) Policy() ShardPolicy {
	return NumRange
}

// Parse implements Configurer
func (p *NumRangeRule) Parse() error {
	if len(p.Ranges) == 0 {
		return fmt.Errorf("invalid ranges")
	}

	for _, v := range p.Ranges {
		if v.Begin > v.End {
			return fmt.Errorf("invalid range begin:%d > end:%d", v.Begin, v.End)
		}
	}
	sort.Slice(p.Ranges, func(i, j int) bool {
		return p.Ranges[i].Begin < p.Ranges[j].Begin
	})

	for i := 1; i < len(p.Ranges); i++ {
		if p.Ranges[i].Begin <= p.Ranges[i-1].End {
			return fmt.Errorf("invalid range[%d].Begin <= range[%d].End", i, i-1)
		}
	}
	return nil
}

// Shard implements ShardRule.Shard
func (p *NumRangeRule) Shard(val interface{}) (shardName string, err error) {
	valInt64, err := c.Int64(val)
	if err != nil {
		return
	}
	count := len(p.Ranges)
	n := sort.Search(count, func(i int) bool {
		return p.Ranges[i].Begin <= valInt64 && p.Ranges[i].End >= valInt64
	})
	if n < count {
		return p.Ranges[n].Name, nil
	}
	if p.DefaultName != "" {
		return p.DefaultName, nil
	}
	return "", fmt.Errorf("can't find name for val %d", val)
}

// ShardFieldName 用于分片的字段名
func (p *NumRangeRule) ShardFieldName() string {
	return p.FieldName
}

// OneRule 选择一个
type OneRule struct {
	Hash     *HashRule     `yaml:"hash"`
	Named    *NamedRule    `yaml:"named"`
	NumRange *NumRangeRule `yaml:"num_range"`
	policy   ShardPolicy
}

// Parse implements Configurer
func (p *OneRule) Parse() error {
	var rules = []ShardRule{p.Hash, p.Named, p.NumRange}
	for _, v := range rules {
		if v == nil {
			continue
		}
		if err := v.Parse(); err != nil {
			return err
		}
		if p.policy == "" {
			p.policy = v.Policy()
		} else {
			return fmt.Errorf("only allow one rule")
		}
	}

	if p.policy == "" {
		return fmt.Errorf("no rule")
	}
	return nil
}

// Policy implements ShardPolicy.Policy
func (p *OneRule) Policy() ShardPolicy {
	return p.policy
}

// Shard implements ShardPolicy.Shard
func (p *OneRule) Shard(val interface{}) (shardName string, err error) {
	rule, err := p.rule()
	if err != nil {
		return
	}
	return rule.Shard(val)
}

// ShardFieldName 用于分片的字段名
func (p *OneRule) ShardFieldName() string {
	rule, _ := p.rule()
	return rule.ShardFieldName()
}

func (p *OneRule) rule() (ShardRule, error) {
	switch p.policy {
	case Hash:
		return p.Hash, nil
	case Named:
		return p.Named, nil
	case NumRange:
		return p.NumRange, nil
	default:
		return nil, fmt.Errorf("unsupport shard policy:%s", p.policy)
	}
}
