package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type aService struct {
	BaseService
}

type bService struct {
	BaseService
}

func TestServices(t *testing.T) {
	as := &aService{BaseService{SName: "a", Order: -1}}
	bs := &bService{BaseService{SName: "b", Order: 2}}
	s1 := NewServices([]Service{as, bs}, true)
	assert.Equal(t, 2, len(s1.sorted))
	assert.Equal(t, true, s1.Init())
	assert.Equal(t, true, s1.Start())
	assert.Equal(t, true, s1.Stop())

	as.state = NEW
	bs.state = NEW

	s2 := NewServices([]Service{bs, as}, true)
	assert.Equal(t, 2, len(s2.sorted))
	requiredOrder := []string{"a", "b"}
	for i := 0; i < len(s2.sorted); i++ {
		assert.Equal(t, requiredOrder[i], s2.sorted[i].Name())
	}

	s2 = NewServices([]Service{bs, as}, false)
	assert.Equal(t, 2, len(s2.sorted))
	requiredOrder = []string{"b", "a"}
	for i := 0; i < len(s2.sorted); i++ {
		assert.Equal(t, requiredOrder[i], s2.sorted[i].Name())
	}

	assert.Equal(t, true, s2.Init())
	assert.Equal(t, true, s2.Start())
	assert.Equal(t, true, s2.Stop())
}

type failingStopService struct{ BaseService }

func (p *failingStopService) Stop() bool { return false }

type observedStopService struct {
	BaseService
	stopped bool
}

func (p *observedStopService) Stop() bool { p.stopped = true; return true }

func TestServicesReportStopFailuresAndContinue(t *testing.T) {
	failed := &failingStopService{}
	other := &observedStopService{}
	services := NewServices([]Service{failed, other}, false)
	assert.True(t, services.Init())
	assert.True(t, services.Start())
	assert.False(t, services.Stop())
	assert.Equal(t, FAILED, failed.State())
	assert.True(t, other.stopped)
	assert.Equal(t, TERMINATED, other.State())
}
