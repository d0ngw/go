package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	SetLogLevel(Debug)
	Debugf("this is a test")
	assert.True(t, DebugEnabled())
	assert.True(t, InfoEnabled())
	SetLogLevel(Info)
	assert.False(t, DebugEnabled())
	assert.True(t, InfoEnabled())
	Debugf("this is a test, no debug")
	Infof("this is a test, info")
	SetLogLevel(0)
	assert.False(t, DebugEnabled())
	assert.True(t, InfoEnabled())
	Infof("this is a test, no level")
	Logf(Warn, "The is a test, warn")
	SetLogLevel(Error)
	assert.False(t, DebugEnabled())
	assert.False(t, InfoEnabled())
	assert.True(t, ErrorEnabled())
	Infof("this is a test, no error")
	Errorf("this is a test, error")
}
