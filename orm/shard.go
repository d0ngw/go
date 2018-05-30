package orm

import (
	"fmt"
	"strconv"
)

// ShardPolicy 分片规则
type ShardPolicy string

// ShardRule 分片规则的实现
type ShardRule interface {
	// Policy 返回策略名称
	Policy() ShardPolicy
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
	Count      int64  `yaml:"count"`
	NamePrefix string `yaml:"name_prefix"`
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
	return nil
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

// NumRangeRule 数字区间
type NumRangeRule struct {
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
	//sort.Slice(p.Ranges,)
	return nil
}

// BuildHashShardFunc 构建Hash函数
func BuildHashShardFunc(hashRule *HashRule, valFunc func() int64) (f func() (string, error), err error) {
	if hashRule == nil || valFunc == nil {
		err = fmt.Errorf("invalid shard params")
		return
	}
	f = func() (string, error) {
		val := valFunc()
		if val < 0 {
			return "", fmt.Errorf("invalid hash val %d", val)
		}
		return hashRule.NamePrefix + strconv.FormatInt((val%hashRule.Count), 10), nil
	}
	return
}

// BuildNamedShardFunc 构建Named函数
func BuildNamedShardFunc(namedRule *NamedRule) (f func() (string, error), err error) {
	if namedRule == nil {
		err = fmt.Errorf("invalid nameRule")
		return
	}
	f = func() (string, error) {
		return namedRule.Name, nil
	}
	return
}

// BuildNumRangeShardFunc 构建Num Range函数
func BuildNumRangeShardFunc(numRangeRule *NumRangeRule, valFunc func() int64) (f func() (string, error), err error) {
	if numRangeRule == nil {
		err = fmt.Errorf("invalid numRangeRule")
		return
	}
	if valFunc == nil {
		err = fmt.Errorf("invalid valFunc")
		return
	}

	f = func() (string, error) {
		val := valFunc()
		for _, v := range numRangeRule.Ranges {
			if val >= v.Begin && val <= v.End {
				return v.Name, nil
			}
		}
		if numRangeRule.DefaultName != "" {
			return numRangeRule.DefaultName, nil
		}
		return "", fmt.Errorf("can't find name for val %d", val)
	}
	return
}
