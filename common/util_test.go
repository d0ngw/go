package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"runtime"
	"syscall"
	"testing"
	"time"
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
