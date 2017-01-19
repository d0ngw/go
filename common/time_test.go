package common

import (
	"fmt"
	"testing"
)

func TestUnixMillisTime(t *testing.T) {
	tt := UnixMillsTime(1453839313078)
	fmt.Println(tt)
	fmt.Println(tt.Year(), tt.Month(), tt.Day(), tt.Hour())
}
