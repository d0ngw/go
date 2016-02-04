package redis

import (
	"bytes"
	"encoding/binary"
)

//uint32 Hash函数
type HashFuncU32 func(data []byte) uint32

//uint64 Hash函数
type HashFuncU64 func(data []byte) uint64

//int64 Hash函数
type HashFuncI64 func(data []byte) int64

//MurmurHash2,32位版本,标准版
func MurmurHash32(data []byte, seed uint32) uint32 {
	const m uint32 = 0x5bd1e995
	const r uint8 = 24

	var length uint32 = uint32(len(data))
	var h uint32 = seed ^ length

	nblocks := int(length / 4)
	buf := bytes.NewBuffer(data)
	for i := 0; i < nblocks; i++ {
		var k uint32
		_ = binary.Read(buf, binary.LittleEndian, &k)
		k *= m
		k ^= k >> r
		k *= m

		h *= m
		h ^= k
	}

	tailIndex := nblocks * 4
	switch length & 3 {
	case 3:
		h ^= uint32(data[tailIndex+2]) << 16
		fallthrough
	case 2:
		h ^= uint32(data[tailIndex+1]) << 8
		fallthrough
	case 1:
		h ^= uint32(data[tailIndex])
		h *= m
	}

	h ^= h >> 13
	h *= m
	h ^= h >> 15
	return h
}

//MurmurHash2,32位版本,与Nginx Lua的版本兼容(Nginx Lua的seed = 0)
func MurmurHash32_Nginx(data []byte) uint32 {
	return MurmurHash32(data, 0)
}

//MurmurHash2,64位版本,标准版
func MurmurHash64A(data []byte, seed uint64) uint64 {
	const m uint64 = 0xc6a4a7935bd1e995
	const r uint8 = 47

	var length = uint64(len(data))
	var h uint64 = seed ^ (length * m)

	nblocks := int(length / 8)
	buf := bytes.NewBuffer(data)
	for i := 0; i < nblocks; i++ {
		var k uint64
		_ = binary.Read(buf, binary.LittleEndian, &k)
		k *= m
		k ^= k >> r
		k *= m

		h ^= k
		h *= m
	}

	tailIndex := nblocks * 8
	switch length & 7 {
	case 7:
		h ^= uint64(data[tailIndex+6]) << 48
		fallthrough
	case 6:
		h ^= uint64(data[tailIndex+5]) << 40
		fallthrough
	case 5:
		h ^= uint64(data[tailIndex+4]) << 32
		fallthrough
	case 4:
		h ^= uint64(data[tailIndex+3]) << 24
		fallthrough
	case 3:
		h ^= uint64(data[tailIndex+2]) << 16
		fallthrough
	case 2:
		h ^= uint64(data[tailIndex+1]) << 8
		fallthrough
	case 1:
		h ^= uint64(data[tailIndex])
		h *= m
	}

	h ^= h >> r
	h *= m
	h ^= h >> r
	return h
}

//MurmurHash2,64位版本,与Jedis版本兼容
func MurmurHash64A_Jedis(data []byte, seed int64) int64 {
	var um uint64 = 0xc6a4a7935bd1e995
	var m int64 = int64(um)
	const r uint8 = 47

	var length = int64(len(data))
	var h int64 = seed ^ (length * m)

	nblocks := int(length / 8)
	buf := bytes.NewBuffer(data)
	for i := 0; i < nblocks; i++ {
		var k int64
		_ = binary.Read(buf, binary.LittleEndian, &k)
		k *= m
		k ^= int64(uint64(k) >> r)
		k *= m

		h ^= k
		h *= m
	}

	tailIndex := nblocks * 8
	switch length & 7 {
	case 7:
		h ^= int64(data[tailIndex+6]) << 48
		fallthrough
	case 6:
		h ^= int64(data[tailIndex+5]) << 40
		fallthrough
	case 5:
		h ^= int64(data[tailIndex+4]) << 32
		fallthrough
	case 4:
		h ^= int64(data[tailIndex+3]) << 24
		fallthrough
	case 3:
		h ^= int64(data[tailIndex+2]) << 16
		fallthrough
	case 2:
		h ^= int64(data[tailIndex+1]) << 8
		fallthrough
	case 1:
		h ^= int64(data[tailIndex])
		h *= m
	}

	h ^= int64(uint64(h) >> r)
	h *= m
	h ^= int64(uint64(h) >> r)
	return h
}
