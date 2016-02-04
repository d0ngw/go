package redis

import (
	"fmt"
	"testing"
)

func TestM(t *testing.T) {
	var um uint64 = 0xc6a4a7935bd1e995
	var m int64 = int64(um)
	fmt.Println(um)
	fmt.Println(um >> 2)
	fmt.Println(m)
	fmt.Println(uint64(m) >> 2)
}

func ExampleMurmurHash32() {
	data := "123456"
	h := MurmurHash32([]byte(data), 10)
	fmt.Println(h)
	h = MurmurHash32([]byte(data), 11)
	fmt.Println(h)
	h = MurmurHash32([]byte(data), 0x1234ABCD)
	fmt.Println(h)
	// Output:
	// 3957618599
	// 1027164520
	// 4286601330
}

func ExampleMurmurHash64A() {
	h := MurmurHash64A([]byte("123456"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A([]byte("1"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A([]byte(""), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A([]byte("1234"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A([]byte("1234567"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A([]byte("12345678"), 0x1234ABCD)
	fmt.Println(h)
	// Output:
	// 4309972499350366586
	// 1836011592861486120
	// 8371356515094919947
	// 74063912950350678
	// 7948810892520069100
	// 5197521178503088135
}

func ExampleMurmurHash64A_Jedis() {
	h := MurmurHash64A_Jedis([]byte("123456"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A_Jedis([]byte("1"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A_Jedis([]byte("123"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A_Jedis([]byte(""), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A_Jedis([]byte("1234"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A_Jedis([]byte("12345"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A_Jedis([]byte("1234567"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A_Jedis([]byte("12345678"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A_Jedis([]byte("123abc"), 0x1234ABCD)
	fmt.Println(h)
	h = MurmurHash64A_Jedis([]byte("i0"), 0x1234ABCD)
	fmt.Println(h)

	// Output:
	// 4309972499350366586
	// 1836011592861486120
	// -530043478380740601
	// 8371356515094919947
	// 74063912950350678
	// -4562369574196896169
	// 7948810892520069100
	// 5197521178503088135
	// 8020980183221229760
	// -3159727732362933533
}

func ExampleMurmurHash32_Empty() {
	data := ""
	h := MurmurHash32([]byte(data), 10)
	fmt.Println(h)
	h = MurmurHash32([]byte(data), 11)
	fmt.Println(h)
	h = MurmurHash32([]byte(data), 0x1234ABCD)
	fmt.Println(h)
	// Output:
	// 2519872436
	// 4060409197
	// 3363255909
}

func ExampleMurmurHash32_Nginx() {
	data := []string{"", "1234567"}
	for _, d := range data {
		h := MurmurHash32_Nginx([]byte(d))
		fmt.Println(h)
	}
	// Output:
	// 0
	// 2438402682
}
