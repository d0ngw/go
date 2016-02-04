package common

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

type aService struct {
	BaseService
}

type bService struct {
	BaseService
}

type alpahName []Service

func (a alpahName) Len() int {
	return len(a)
}

func (a alpahName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a alpahName) Less(i, j int) bool {
	return a[i].Name() < a[j].Name()
}

func alphaSorter(ss []Service) sort.Interface {
	toSort := alpahName(ss)
	sort.Sort(toSort)
	return toSort
}

func TestServices(t *testing.T) {
	as := &aService{BaseService{SName: "a"}}
	bs := &bService{BaseService{SName: "b"}}
	s1 := NewServices([]Service{bs, as}, nil)
	assert.Equal(t, 2, len(s1.sorted))
	assert.Equal(t, true, s1.Init())
	assert.Equal(t, true, s1.Start())
	assert.Equal(t, true, s1.Stop())

	as.state = NEW
	bs.state = NEW
	s2 := NewServices([]Service{bs, as}, alphaSorter)
	assert.Equal(t, 2, len(s2.sorted))
	requiredOrder := []string{"a", "b"}
	for i := 0; i < len(s2.sorted); i++ {
		assert.Equal(t, requiredOrder[i], s2.sorted[i].Name())
	}
	assert.Equal(t, true, s2.Init())
	assert.Equal(t, true, s2.Start())
	assert.Equal(t, true, s2.Stop())
}
