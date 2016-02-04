package common

import (
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

func TestNotEmpty(t *testing.T) {
	va := &NotEmptyValidator{}
	assert.False(t, va.Validate(""))
	assert.False(t, va.Validate(" "))
	assert.False(t, va.Validate("　"))
	assert.True(t, va.Validate(" abc "))
	assert.True(t, va.Validate("　a　"))
}

func TestInteger(t *testing.T) {
	va32 := &Int32Validator{
		min: -3,
		max: 10}
	assert.False(t, va32.Validate("a"))
	assert.False(t, va32.Validate("11"))
	assert.True(t, va32.Validate("10"))
	assert.True(t, va32.Validate("-3"))

	va64 := &Int32Validator{
		min: -3,
		max: 10}
	assert.False(t, va64.Validate("a"))
	assert.False(t, va64.Validate("11"))
	assert.True(t, va64.Validate("10"))
	assert.True(t, va64.Validate("-3"))
}

func TestFloat(t *testing.T) {
	va32 := &Float32Validator{
		min: 0.1,
		max: 10}
	assert.False(t, va32.Validate("a"))
	assert.False(t, va32.Validate("11"))
	assert.True(t, va32.Validate("10"))
	assert.False(t, va32.Validate("-3"))

	va64 := &Float64Validator{
		min: -3,
		max: 10.5}
	assert.False(t, va64.Validate("a"))
	assert.False(t, va64.Validate("11"))
	assert.True(t, va64.Validate("10.4"))
	assert.True(t, va64.Validate("-3"))
}

func TestRegex(t *testing.T) {
	rv := &RegExValidator{
		pattern: regexp.MustCompile("^a+")}

	assert.True(t, rv.Validate("a"))
	assert.False(t, rv.Validate("1a"))
}
