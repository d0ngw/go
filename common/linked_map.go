package common

import (
	"container/list"
	"sync"
)

type mapElement struct {
	val     interface{}
	element *list.Element
}

// LinkedMap implements linked map
type LinkedMap struct {
	mutex sync.RWMutex
	l     *list.List
	m     map[interface{}]*mapElement
}

// NewLinkedMap create linked map
func NewLinkedMap() *LinkedMap {
	return &LinkedMap{
		l: list.New(),
		m: map[interface{}]*mapElement{},
	}
}

// Put put value with key
func (p *LinkedMap) Put(key, value interface{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	var keyElem *list.Element
	if pre, ok := p.m[key]; !ok {
		keyElem = p.l.PushBack(key)
		p.m[key] = &mapElement{
			val:     value,
			element: keyElem,
		}
	} else {
		pre.val = value
	}
}

// Get value with key
func (p *LinkedMap) Get(key interface{}) (val interface{}, ok bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if pre, ok := p.m[key]; ok {
		return pre.val, ok
	}
	return nil, false
}

// Remove value with key
func (p *LinkedMap) Remove(key interface{}) (preVal interface{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if pre, ok := p.m[key]; ok {
		delete(p.m, key)
		p.l.Remove(pre.element)
		return pre.val
	}
	return nil
}

// Len return the length of the map
func (p *LinkedMap) Len() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.l.Len()
}

// MapEntry define map entry with key and value
type MapEntry struct {
	Key   interface{}
	Value interface{}
}

// Entries return entry slice
func (p *LinkedMap) Entries() []*MapEntry {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	entries := make([]*MapEntry, p.l.Len())
	var i = 0
	for e := p.l.Front(); e != nil; e = e.Next() {
		key := e.Value
		value := p.m[key].val
		entries[i] = &MapEntry{Key: key, Value: value}
		i++
	}
	return entries
}
