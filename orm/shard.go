package orm

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"

	c "github.com/d0ngw/go/common"
)

// ShardPolicy identifies a sharding policy.
type ShardPolicy string

// ShardRule defines a sharding rule.
type ShardRule interface {
	c.Configurer
	// Policy returns the policy name.
	Policy() ShardPolicy
	// Shard computes the shard name.
	Shard(val interface{}) (shardName string, err error)
	// ShardFieldName returns the field used for sharding.
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

// IsValid reports whether the rule is valid.
func (p ShardPolicy) IsValid() bool {
	return p == Hash || p == Named || p == NumRange
}

// HashRule selects a shard using a hash.
type HashRule struct {
	Count      int64  `yaml:"count"`       // Number of hash buckets.
	NamePrefix string `yaml:"name_prefix"` // Shard name prefix.
	FieldName  string `yaml:"field_name"`  // Field used to compute the hash.
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

// ShardFieldName returns the field used for sharding.
func (p *HashRule) ShardFieldName() string {
	return p.FieldName
}

// NamedRule selects a shard by name.
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

// ShardFieldName returns the field used for sharding.
func (p *NamedRule) ShardFieldName() string {
	return ""
}

// NumRangeRule selects a shard by numeric range.
type NumRangeRule struct {
	FieldName   string `yaml:"field_name"`   // Field used to select a shard.
	DefaultName string `yaml:"default_name"` // Default shard name.
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
			return fmt.Errorf("invalid range[%d].Begin %d <= range[%d].End %d", i, p.Ranges[i].Begin, i-1, p.Ranges[i-1].End)
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

	i, j, found := 0, len(p.Ranges), -1

	for i < j {
		h := int(uint(i+j) >> 1)
		r := p.Ranges[h]
		if r.Begin <= valInt64 && r.End >= valInt64 {
			found = h
			break
		} else if r.Begin < valInt64 {
			i = h + 1
		} else if r.Begin > valInt64 {
			j = h
		}
	}
	if found >= 0 {
		return p.Ranges[found].Name, nil
	}
	if p.DefaultName != "" {
		return p.DefaultName, nil
	}
	return "", fmt.Errorf("can't find name for val %d", val)
}

// ShardFieldName returns the field used for sharding.
func (p *NumRangeRule) ShardFieldName() string {
	return p.FieldName
}

// OneRule selects one sharding rule.
type OneRule struct {
	Hash     *HashRule     `yaml:"hash"`
	Named    *NamedRule    `yaml:"named"`
	NumRange *NumRangeRule `yaml:"num_range"`
	policy   ShardPolicy
	rule     ShardRule
}

// Parse implements Configurer
func (p *OneRule) Parse() error {
	var rules = []ShardRule{p.Hash, p.Named, p.NumRange}
	for _, v := range rules {
		if v == nil || reflect.ValueOf(v).IsNil() {
			continue
		}
		if err := v.Parse(); err != nil {
			return err
		}
		if p.policy == "" {
			p.policy = v.Policy()
			p.rule = v
		} else {
			return fmt.Errorf("only allow one rule")
		}
	}

	if p.policy == "" || p.rule == nil {
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
	return p.rule.Shard(val)
}

// ShardFieldName returns the field used for sharding.
func (p *OneRule) ShardFieldName() string {
	return p.rule.ShardFieldName()
}
