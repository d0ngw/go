package common

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type opType uint

//opType的类型
const (
	opPut           opType = iota //添加
	opPutOnlyAbsent               //添加,如果指定的key已经存在,则返回error
	opDel                         //删除指定的key
)

type cowMap map[interface{}]interface{}

//CopyOnWriteMap copy on write map
type CopyOnWriteMap struct {
	m     atomic.Value
	mutex sync.Mutex
}

// NewCopyOnWriteMap 创建CopyOnWriteMap
func NewCopyOnWriteMap() *CopyOnWriteMap {
	reg := &CopyOnWriteMap{}
	reg.m.Store(make(cowMap))
	return reg
}

// copyMap 复制src map
func copyMap(src cowMap) cowMap {
	m := make(cowMap)
	for k, v := range src {
		m[k] = v
	}
	return m
}

// modify 根据opType的操作类型修改CopyOnWriteMap
func (p *CopyOnWriteMap) modify(key interface{}, value interface{}, op opType) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	m1 := p.m.Load().(cowMap)

	switch op {
	case opPutOnlyAbsent:
		if _, ok := m1[key]; ok {
			return fmt.Errorf("Duplicate key:%v", key)
		}
		fallthrough
	case opPut:
		m2 := copyMap(m1)
		m2[key] = value
		p.m.Store(m2)
	case opDel:
		m2 := copyMap(m1)
		delete(m2, key)
		p.m.Store(m2)
	default:
		panic(fmt.Errorf("Unsupported op type %#v", op))
	}

	return nil
}

// Put key及对应的value,如果key已经存在,则进行替换
func (p *CopyOnWriteMap) Put(key interface{}, value interface{}) {
	p.modify(key, value, opPut)
}

// PutIfAbsent put key及对应的value,如果key已经存在,不进行替换,并返回错误
func (p *CopyOnWriteMap) PutIfAbsent(key interface{}, value interface{}) error {
	return p.modify(key, value, opPutOnlyAbsent)
}

// Delete 删除key
func (p *CopyOnWriteMap) Delete(key interface{}) {
	p.modify(key, nil, opDel)
}

//Get 取得key对应的值
func (p *CopyOnWriteMap) Get(key interface{}) interface{} {
	m1 := p.m.Load().(cowMap)
	if value, ok := m1[key]; ok {
		return value
	}
	return nil
}

// CopyOnWriteSlice
type cowSlice []interface{}

// CopyOnWriteSlice copy on write slice
type CopyOnWriteSlice struct {
	m     atomic.Value
	mutex sync.Mutex
}

// NewCopyOnWriteSlice 创建CopyOnWriteSlice
func NewCopyOnWriteSlice() *CopyOnWriteSlice {
	reg := &CopyOnWriteSlice{}
	reg.m.Store(make(cowSlice, 0))
	return reg
}

// modify 根据opType的类型,修改CopyOnWriteSlice
func (p *CopyOnWriteSlice) modify(value interface{}, op opType) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	m1 := p.m.Load().(cowSlice)

	switch op {
	case opPut:
		m2 := make(cowSlice, len(m1), len(m1)+1)
		copy(m2, m1)
		m2 = append(m2, value)
		p.m.Store(m2)
	case opDel:
		m2 := make(cowSlice, 0, len(m1))
		for _, v := range m1 {
			if v != value {
				m2 = append(m2, v)
			}
		}
		p.m.Store(m2)
	default:
		panic(fmt.Errorf("Unsupported op type %#v", op))
	}

	return nil
}

// Add 添加
func (p *CopyOnWriteSlice) Add(value interface{}) error {
	if value == nil {
		panic("Can't add nil value")
	}
	return p.modify(value, opPut)
}

// Delete 删除value
func (p *CopyOnWriteSlice) Delete(value interface{}) error {
	if value == nil {
		panic("Can't delete nil value")
	}
	return p.modify(value, opDel)
}

// Get 取得Slice
func (p *CopyOnWriteSlice) Get() []interface{} {
	return p.m.Load().(cowSlice)
}
