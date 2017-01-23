package common

import (
	"fmt"
	"testing"
	"time"
)

func TestUnixMillisTime(t *testing.T) {
	tt := UnixMillsTime(1453839313078)
	fmt.Println(tt)
	fmt.Println(tt.Year(), tt.Month(), tt.Day(), tt.Hour())
}

func TestUnixMillis(t *testing.T) {
	now := time.Now()
	fmt.Println(now.UnixNano())
	fmt.Println(UnixMills(now))
}
