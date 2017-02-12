package common

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type st struct {
}

func TestGetType(t *testing.T) {
	vt := GetFirstFieldType(struct{ st *interface{} }{})
	assert.Equal(t, vt.Kind(), reflect.Ptr)
	assert.Equal(t, vt.Elem().Kind(), reflect.Interface)
}

func TestShutdownHook(t *testing.T) {
	shook := NewShutdownhook()
	shook.AddHook(func() {
		_, c1, c2, _ := runtime.Caller(1)
		fmt.Println("Call @", c1, c2)
	})

	fmt.Println("First wait")
	go func() {
		time.Sleep(time.Duration(100) * time.Millisecond)
		shook.ch <- syscall.SIGINT
	}()
	shook.WaitShutdown()
}

func TestInt64(t *testing.T) {
	testInt64(t, 2, 2)
	testInt64(t, int8(2), 2)
	testInt64(t, int16(2), 2)
	testInt64(t, int32(2), 2)
	testInt64(t, int64(2), 2)
	testInt64(t, "2", 2)
	testInt64(t, float32(2.0), 2)
	testInt64(t, float64(2.0), 2)
	testInt64(t, json.Number("2"), 2)
	_, err := Int64(struct{}{})
	assert.NotNil(t, err)
}

func testInt64(t *testing.T, v interface{}, av int64) {
	i, err := Int64(v)
	assert.Nil(t, err)
	assert.EqualValues(t, i, av)
}

func TestFloat64(t *testing.T) {
	testFloat64(t, 2, 2.0)
	testFloat64(t, int8(2), 2.0)
	testFloat64(t, int16(2), 2.0)
	testFloat64(t, int32(2), 2.0)
	testFloat64(t, int64(2), 2.0)
	testFloat64(t, "2", 2.0)
	testFloat64(t, float32(2.0), 2.0)
	testFloat64(t, float64(2.0), 2.0)
	testFloat64(t, json.Number("2"), 2.0)
	_, err := Float64(struct{}{})
	assert.NotNil(t, err)
}

func testFloat64(t *testing.T, v interface{}, av float64) {
	i, err := Float64(v)
	assert.Nil(t, err)
	assert.EqualValues(t, i, av)
}
