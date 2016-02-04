package orm

import (
	"fmt"
	"testing"
	"unsafe"
)

var (
	config = MysqlDBConfig{
		"root",
		"",
		"127.0.0.1:3306",
		"test",
		100,
		10}
	dbp, err = config.CreateDBPool()
)

type person struct {
	Name string
	Age  uint8
}

func TestPointer(t *testing.T) {
	person := person{"a", 1}
	fmt.Printf("%v\n", person)
	pp := &person
	fmt.Printf("%p\n", pp)
	puptr := uintptr(unsafe.Pointer(pp))
	fmt.Printf("%x\n", puptr)
	namePtr := puptr + unsafe.Offsetof(pp.Name)
	agePtr := puptr + unsafe.Offsetof(pp.Age)
	fmt.Printf("namePtr:%x,agePtr:%x\n", namePtr, agePtr)
	pp.Name = "ab"
	namePtr = puptr + unsafe.Offsetof(pp.Name)
	agePtr = puptr + unsafe.Offsetof(pp.Age)
	fmt.Printf("namePtr:%x,agePtr:%x\n", namePtr, agePtr)
}

func TestPointer2(t *testing.T) {
	p2 := new(person)
	checkAddress(p2, "p2")
	fmt.Printf("%s address:%p\n", "p2 add", p2)
	checkAddress(*p2, "p2 value")
	fmt.Printf("p2:%p\n", p2)
	is := make([]int, 0)
	checkAddress(is, "is")
	checkAddress(&is, "is ptr")
	fmt.Printf("%T,%T\n", is, &is)
	is = append(*&is, 1)
	fmt.Printf("%v,%v\n", is, &is)

	m := make(map[*string]int)
	a := "a"
	a2 := "a"
	m[&a] = 1
	fmt.Printf("m:%v\n", m[&a])
	fmt.Printf("m:%v\n", m[&a2])
	checkAddress(m, "m")
	ss := "中国bc"
	for i, v := range ss {
		fmt.Printf("i:%v,v:%v,vt:%T", i, v, v)
	}
}

func checkAddress(e interface{}, name string) {
	fmt.Printf("%s address:%p\n", name, &e)
}
