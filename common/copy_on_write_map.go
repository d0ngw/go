package common

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type opType uint

// Supported copy-on-write operations.
const (
	opPut           opType = iota // Add a value.
	opPutOnlyAbsent               // Add a value, returning an error if the key exists.
	opDel                         // Delete the specified key.
)

type cowMap map[interface{}]interface{}

// CopyOnWriteMap copy on write map
type CopyOnWriteMap struct {
	m     atomic.Value
	mutex sync.Mutex
}

// NewCopyOnWriteMap creates a copy-on-write map.
func NewCopyOnWriteMap() *CopyOnWriteMap {
	reg := &CopyOnWriteMap{}
	reg.m.Store(make(cowMap))
	return reg
}

// copyMap copies src.
func copyMap(src cowMap) cowMap {
	m := make(cowMap)
	for k, v := range src {
		m[k] = v
	}
	return m
}

// modify applies an operation to a copy of the map.
func (p *CopyOnWriteMap) modify(key interface{}, value interface{}, op opType) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	m1 := p.m.Load().(cowMap)

	switch op {
	case opPutOnlyAbsent:
		if _, ok := m1[key]; ok {
			return fmt.Errorf("duplicate key:%v", key)
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
		panic(fmt.Errorf("unsupported op type %#v", op))
	}

	return nil
}

// Put stores value under key, replacing any existing value.
func (p *CopyOnWriteMap) Put(key interface{}, value interface{}) {
	p.modify(key, value, opPut)
}

// PutIfAbsent stores value unless key already exists, in which case it returns an error.
func (p *CopyOnWriteMap) PutIfAbsent(key interface{}, value interface{}) error {
	return p.modify(key, value, opPutOnlyAbsent)
}

// Delete removes key.
func (p *CopyOnWriteMap) Delete(key interface{}) {
	p.modify(key, nil, opDel)
}

// Get returns the value associated with key.
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

// NewCopyOnWriteSlice creates a copy-on-write slice.
func NewCopyOnWriteSlice() *CopyOnWriteSlice {
	reg := &CopyOnWriteSlice{}
	reg.m.Store(make(cowSlice, 0))
	return reg
}

// modify applies an operation to a copy of the slice.
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
		panic(fmt.Errorf("unsupported op type %#v", op))
	}

	return nil
}

// Add appends a value.
func (p *CopyOnWriteSlice) Add(value interface{}) error {
	if value == nil {
		panic("Can't add nil value")
	}
	return p.modify(value, opPut)
}

// Delete removes value.
func (p *CopyOnWriteSlice) Delete(value interface{}) error {
	if value == nil {
		panic("Can't delete nil value")
	}
	return p.modify(value, opDel)
}

// Get returns the stored slice.
func (p *CopyOnWriteSlice) Get() []interface{} {
	return p.m.Load().(cowSlice)
}
