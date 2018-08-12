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

type TestStruct struct {
	ID int64
}

func TestByteSlice2String(t *testing.T) {
	var bs []byte
	t.Logf("bs:%v", bs)

	str0 := ByteSlice2String(bs)
	str1 := string(bs)
	assert.Equal(t, str0, str1)
	t.Logf("str0:`%s`,str1:`%s`,%p", str0, str1, &str1)

	var bss = []byte("abcdefgh")
	str1 = string(bss)
	str2 := "1"
	bs0 := String2ByteSlice(str1)
	bs0[0] = 'A'

	fmt.Printf("str1:%p,%s\n", &str1, str1)
	assert.EqualValues(t, len(str1), len(bs0))
	t.Logf("%s,%s,%d,%p", str1, string(bs0), bs0[0], &bs0)
	bs0[0] = 1

	bs1 := String2ByteSlice(str2)
	t.Logf("str2:%p,bs1:%v,%p,cap:%d", &str2, bs1, &bs1, cap(bs1))
}

type CopyBase0 struct {
	ID2         int32
	BaseCountry int8
	Haha        string
}

type CopyBase struct {
	CopyBase0
	BaseName string
}

type CopyBase2 struct {
	CopyBase
	BaseName2 string
}

func TestStructCopier(t *testing.T) {
	var from = &struct {
		ID      int64
		Name    string
		Age     int32
		Address []string
		T       *TestStruct
		CopyBase2
	}{
		ID:      1,
		Name:    "ok",
		Age:     32,
		Address: []string{"a", "b", "c"},
		T:       &TestStruct{ID: 100},
	}
	from.BaseName = "b1"
	from.BaseName2 = "b2"
	from.BaseCountry = 10

	var to = &struct {
		ID2  int32
		Name string
		TestStruct
		Age         int32
		Address     []string
		T           *TestStruct
		BaseName    string
		BaseName2   string
		BaseCountry int8
		Haha        string
	}{}

	copier, err := NewStructCopier(from, to)
	assert.NoError(t, err)
	err = copier(from, to)
	assert.NoError(t, err)
	assert.Equal(t, from.ID, to.ID)
	assert.Equal(t, from.Name, to.Name)
	assert.Equal(t, from.Age, to.Age)
	assert.Equal(t, from.Address, to.Address)
	assert.Equal(t, from.T.ID, to.T.ID)
	assert.Equal(t, from.BaseName, to.BaseName)
	assert.Equal(t, from.BaseName2, to.BaseName2)
	assert.Equal(t, from.BaseCountry, to.BaseCountry)
	t.Log(to.Address)
}

func TestIsValNil(t *testing.T) {
	var i int
	assert.False(t, IsValNil(i))

	var ip *int
	assert.True(t, IsValNil(ip))

	var j float32
	assert.False(t, IsValNil(j))

	var bs []byte
	var bsi interface{} = bs
	assert.True(t, bs == nil)
	assert.False(t, bsi == nil)
	assert.True(t, IsValNil(bs))

	var m map[string]string
	assert.True(t, IsValNil(m))
	assert.True(t, m == nil)

	m = map[string]string{"a": "b"}
	assert.False(t, m == nil)
	assert.False(t, IsValNil(m))

	var s string
	assert.False(t, IsValNil(s))

	assert.True(t, IsValNil(nil))
	assert.False(t, IsValNil(struct{}{}))

	var f func()
	assert.True(t, IsValNil(f))

	f = func() {}
	assert.False(t, IsValNil(f))

	var c chan int
	assert.True(t, IsValNil(c))
	c = make(chan int)
	assert.False(t, IsValNil(c))

	var ii interface{}
	assert.True(t, IsValNil(ii))
	ii = ""
	assert.False(t, IsValNil(ii))
}
