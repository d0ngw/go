package common

import (
	"math"
	"unicode/utf16"
)

// JTypeCompatible 定义了和Java类型兼容的接口
type JTypeCompatible interface {
	// HashCode java.lang.Object.hashCode()的实现
	HashCode() int32
}

// JString Java String的兼容类型
type JString string

// HashCode Java String.hashCode的实现
func (p JString) HashCode() int32 {
	var h int32
	if len(p) > 0 {
		runes := []rune(string(p))
		u16 := utf16.Encode(runes)
		for i := 0; i < len(u16); i++ {
			h = 31*h + int32(u16[i])
		}
	}
	return h
}

// JByte Java byte的兼容类型
type JByte int8

// HashCode 计算hashcode
func (p JByte) HashCode() int32 {
	return int32(p)
}

// JShort Java short的兼容类型
type JShort int16

// HashCode 计算hashcode
func (p JShort) HashCode() int32 {
	return int32(p)
}

// JInt Java int的兼容类型
type JInt int32

// HashCode 计算hashcode
func (p JInt) HashCode() int32 {
	return int32(p)
}

// JLong Java long的兼容类型
type JLong int64

// HashCode 计算hashcode
func (p JLong) HashCode() int32 {
	l := int64(p)
	return int32(l ^ int64(uint64(l)>>32))
}

const (
	//FloatConsts中的常量
	fExpBitMask    = 2139095040
	fSignifBitMask = 8388607
	//DoubleConsts中的常量
	dExpBitMask    = 9218868437227405312
	dSignifBitMask = 4503599627370495
)

// floatToIntBits 实现Java中的floatToIntBits
func floatToIntBits(value float32) int32 {
	result := math.Float32bits(value)
	if (result&fExpBitMask == fExpBitMask) && (result&fSignifBitMask != 0) {
		result = 0x7fc00000
	}
	return int32(result)
}

// doubleToLongBits实现Java中的doubleToLongBits
func doubleToLongBits(value float64) int64 {
	result := math.Float64bits(value)
	if (result&dExpBitMask == dExpBitMask) && (result&dSignifBitMask != 0) {
		result = 0x7ff8000000000000
	}
	return int64(result)
}

// JFloat Java float的兼容类型
type JFloat float32

// HashCode float hash code
func (p JFloat) HashCode() int32 {
	return floatToIntBits(float32(p))
}

// JDouble Java double的兼容类型
type JDouble float64

// HashCode double hash code
func (p JDouble) HashCode() int32 {
	return JLong(doubleToLongBits(float64(p))).HashCode()
}
