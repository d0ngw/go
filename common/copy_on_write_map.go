package common

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type opType uint

//opType的类型
const (
	op_put             opType = iota //添加
	op_put_only_absent               //添加,如果指定的key已经存在,则返回error
	op_del                           //删除指定的key
)

//CopyOnWriteMap
type cowMap map[interface{}]interface{}

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
func (self *CopyOnWriteMap) modify(key interface{}, value interface{}, op opType) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	m1 := self.m.Load().(cowMap)

	switch op {
	case op_put_only_absent:
		if _, ok := m1[key]; ok {
			return fmt.Errorf("Duplicate key:%v", key)
		}
		fallthrough
	case op_put:
		m2 := copyMap(m1)
		m2[key] = value
		self.m.Store(m2)
	case op_del:
		m2 := copyMap(m1)
		delete(m2, key)
		self.m.Store(m2)
	default:
		panic(fmt.Errorf("Unsupported op type %#v", op))
	}

	return nil
}

// Put key及对应的value,如果key已经存在,则进行替换
func (self *CopyOnWriteMap) Put(key interface{}, value interface{}) {
	self.modify(key, value, op_put)
}

// Put key及对应的value,如果key已经存在,不进行替换,并返回错误
func (self *CopyOnWriteMap) PutIfAbsent(key interface{}, value interface{}) error {
	return self.modify(key, value, op_put_only_absent)
}

// Delete 删除key
func (self *CopyOnWriteMap) Delete(key interface{}) {
	self.modify(key, nil, op_del)
}

//Get 取得key对应的值
func (self *CopyOnWriteMap) Get(key interface{}) interface{} {
	m1 := self.m.Load().(cowMap)
	if value, ok := m1[key]; ok {
		return value
	} else {
		return nil
	}
}

// CopyOnWriteSlice
type cowSlice []interface{}

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
func (self *CopyOnWriteSlice) modify(value interface{}, op opType) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	m1 := self.m.Load().(cowSlice)

	switch op {
	case op_put:
		m2 := make(cowSlice, len(m1), len(m1)+1)
		copy(m2, m1)
		m2 = append(m2, value)
		self.m.Store(m2)
	case op_del:
		m2 := make(cowSlice, 0, len(m1))
		for _, v := range m1 {
			if v != value {
				m2 = append(m2, v)
			}
		}
		self.m.Store(m2)
	default:
		panic(fmt.Errorf("Unsupported op type %#v", op))
	}

	return nil
}

// Add 添加
func (self *CopyOnWriteSlice) Add(value interface{}) error {
	if value == nil {
		panic("Can't add nil value")
	}
	return self.modify(value, op_put)
}

// Delete 删除value
func (self *CopyOnWriteSlice) Delete(value interface{}) error {
	if value == nil {
		panic("Can't delete nil value")
	}
	return self.modify(value, op_del)
}

// Get 取得Slice
func (self *CopyOnWriteSlice) Get() []interface{} {
	return self.m.Load().(cowSlice)
}
