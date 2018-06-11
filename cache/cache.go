// Package cache 提供缓冲相关的服务
package cache

// Param is the cache param
type Param interface {
	//Group cache group id
	Group() string
	//Key cache key
	Key() string
	//Expire second time
	Expire() int
}

// ParamConf is the cache param conf with cache group,key prefix and expire
type ParamConf struct {
	group     string
	keyPrefix string
	expire    int
}

// NewParamConf create ParamConf
func NewParamConf(group, keyPrefix string, expire int) *ParamConf {
	return &ParamConf{
		group:     group,
		keyPrefix: keyPrefix,
		expire:    expire,
	}
}

// Group return cache group
func (p *ParamConf) Group() string {
	return p.group
}

// Expire return expire second
func (p *ParamConf) Expire() int {
	return p.expire
}

// KeyPrefix return key prefix
func (p *ParamConf) KeyPrefix() string {
	return p.keyPrefix
}

// NewWithExpire create new ParamConf with new expire parameter
func (p *ParamConf) NewWithExpire(expire int) *ParamConf {
	var param = *p
	param.expire = expire
	return &param
}

// NewWithKeyPrefix append keyPrefix to exist ParamConf,return new ParamConf
func (p *ParamConf) NewWithKeyPrefix(keyPrefix string) *ParamConf {
	var param = *p
	param.keyPrefix = p.keyPrefix + keyPrefix
	return &param
}

// NewParamKey create new ParamKey with key
func (p *ParamConf) NewParamKey(key string) *ParamKey {
	return &ParamKey{
		ParamConf: p,
		key:       p.keyPrefix + key,
	}
}

// ParamKey is the cache param with key
type ParamKey struct {
	*ParamConf
	key string
}

// Key implements Param.Key()
func (p *ParamKey) Key() string {
	return p.key
}

// NewWithExpire new key with expire
func (p *ParamKey) NewWithExpire(expire int) *ParamKey {
	var k = *p
	k.expire = expire
	return &k
}
