package common

import "testing"
import "github.com/stretchr/testify/assert"

func TestLinkedMap(t *testing.T) {
	lm := NewLinkedMap()
	for i := 0; i < 100; i++ {
		lm.Put(i, i+1)
	}
	assert.Equal(t, 100, lm.Len())
	for i := 0; i < 100; i++ {
		val, ok := lm.Get(i)
		assert.True(t, ok)
		assert.Equal(t, i+1, val)
	}
	entries := lm.Entries()
	assert.Equal(t, 100, len(entries))
	for i := 0; i < 100; i++ {
		e := entries[i]
		assert.Equal(t, i, e.Key)
		assert.Equal(t, i+1, e.Value)
	}
	for i := 0; i < 100; i++ {
		preVal := lm.Remove(i)
		assert.Equal(t, i+1, preVal)
	}
	assert.Equal(t, 0, lm.Len())
	assert.Equal(t, 0, lm.l.Len())
	assert.Equal(t, 0, len(lm.m))
}
