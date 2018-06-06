package common

import (
	"fmt"
	"hash/fnv"
	"sync"
)

const (
	defaultCurrentMapShardCount = 32
)

type mCurrentMapShared struct {
	items map[interface{}]interface{}
	sync.RWMutex
}

// CurrentMap A "thread" safe map of type interface{}:interface{},auto gen from concurrent_map_template.go
type CurrentMap struct {
	shardCount uint
	shards     []*mCurrentMapShared
}

// NewCurrentMap Create a new concurrent map with 32 shards
func NewCurrentMap() *CurrentMap {
	return NewCurrentMapWithShard(defaultCurrentMapShardCount)
}

// NewCurrentMapWithShard Creates a new concurrent map.
func NewCurrentMapWithShard(shardCount uint) *CurrentMap {
	if shardCount == 0 {
		shardCount = defaultCurrentMapShardCount
	}

	shards := make([]*mCurrentMapShared, shardCount)

	var i uint
	for i = 0; i < shardCount; i++ {
		shards[i] = &mCurrentMapShared{items: make(map[interface{}]interface{})}
	}
	return &CurrentMap{shardCount: shardCount, shards: shards}
}

// Returns shard under given key
func (m *CurrentMap) getShard(key interface{}) *mCurrentMapShared {
	strKey := fmt.Sprintf("%v", key)
	hasher := fnv.New32()
	hasher.Write([]byte(strKey))
	return m.shards[uint(hasher.Sum32())%uint(m.shardCount)]
}

// Set the given value under the specified key.
func (m *CurrentMap) Set(key interface{}, value interface{}) {
	shard := m.getShard(key)
	shard.Lock()
	defer shard.Unlock()
	shard.items[key] = value
}

// SetIfAbsent the given value under the specified key if no value was associated with it.
func (m *CurrentMap) SetIfAbsent(key interface{}, value interface{}) (success bool, preVal interface{}) {
	shard := m.getShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	if !ok {
		shard.items[key] = value
	}
	shard.Unlock()
	return !ok, v
}

// Get Retrieves an element from map under given key.
func (m *CurrentMap) Get(key interface{}) (interface{}, bool) {
	shard := m.getShard(key)
	shard.RLock()
	defer shard.RUnlock()
	val, ok := shard.items[key]
	return val, ok
}

// Count Returns the number of elements within the map.
func (m *CurrentMap) Count() int {
	count := 0
	var i uint
	for i = 0; i < m.shardCount; i++ {
		shard := m.shards[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Has Looks up an item under specified key
func (m *CurrentMap) Has(key interface{}) bool {
	shard := m.getShard(key)
	shard.RLock()
	defer shard.RUnlock()
	_, ok := shard.items[key]
	return ok
}

// Remove an element from the map.
func (m *CurrentMap) Remove(key interface{}) {
	shard := m.getShard(key)
	shard.Lock()
	defer shard.Unlock()
	delete(shard.items, key)
}

// IsEmpty Checks if map is empty.
func (m *CurrentMap) IsEmpty() bool {
	return m.Count() == 0
}

// CurrentMapTuple Used by the Iter & IterBuffered functions to wrap two variables together over a channel,
type CurrentMapTuple struct {
	Key interface{}
	Val interface{}
}

// Iter Returns an iterator which could be used in a for range loop.
func (m CurrentMap) Iter() <-chan CurrentMapTuple {
	ch := make(chan CurrentMapTuple)
	go func() {
		// Foreach shard.
		for _, shard := range m.shards {
			// Foreach key, value pair.
			shard.RLock()
			for key, val := range shard.items {
				ch <- CurrentMapTuple{key, val}
			}
			shard.RUnlock()
		}
		close(ch)
	}()
	return ch
}

// IterBuffered Returns a buffered iterator which could be used in a for range loop.
func (m CurrentMap) IterBuffered() <-chan CurrentMapTuple {
	ch := make(chan CurrentMapTuple, m.Count())
	go func() {
		// Foreach shard.
		for _, shard := range m.shards {
			// Foreach key, value pair.
			shard.RLock()
			for key, val := range shard.items {
				ch <- CurrentMapTuple{key, val}
			}
			shard.RUnlock()
		}
		close(ch)
	}()
	return ch
}

// Items Returns all items as map[interface{}]interface{}
func (m CurrentMap) Items() map[interface{}]interface{} {
	tmp := make(map[interface{}]interface{})

	// Insert items to temporary map.
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}

	return tmp
}
