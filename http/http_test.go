package http

import (
	"fmt"
	"net"
	"testing"
	"unsafe"
)

func TestListner(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println(err)
	}
	if err := ln.Close(); err != nil {
		fmt.Println(err)
	}
	if _, err := ln.Accept(); err != nil {
		fmt.Printf("%T\n", err)
	}
}

func TestChanSize(t *testing.T) {
	fmt.Println(unsafe.Sizeof(make(chan struct{}, 100)))
	fmt.Println(unsafe.Sizeof(make(chan int, 100)))
	i := 9
	var s struct{}
	fmt.Println(unsafe.Sizeof(i))
	fmt.Println(unsafe.Sizeof(s))
}
