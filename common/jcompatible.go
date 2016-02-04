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
func (self JString) HashCode() int32 {
	var h int32
	if len(self) > 0 {
		runes := []rune(string(self))
		u16 := utf16.Encode(runes)
		for i := 0; i < len(u16); i++ {
			h = 31*h + int32(u16[i])
		}
	}
	return h
}

// JByte Java byte的兼容类型
type JByte int8

func (self JByte) HashCode() int32 {
	return int32(self)
}

// JShort Java short的兼容类型
type JShort int16

func (self JShort) HashCode() int32 {
	return int32(self)
}

// JInt Java int的兼容类型
type JInt int32

func (self JInt) HashCode() int32 {
	return int32(self)
}

// JLong Java long的兼容类型
type JLong int64

func (self JLong) HashCode() int32 {
	l := int64(self)
	return int32(l ^ int64(uint64(l)>>32))
}

const (
	//FloatConsts中的常量
	f_EXP_BIT_MASK    = 2139095040
	f_SIGNIF_BIT_MASK = 8388607
	//DoubleConsts中的常量
	d_EXP_BIT_MASK    = 9218868437227405312
	d_SIGNIF_BIT_MASK = 4503599627370495
)

// floatToIntBits 实现Java中的floatToIntBits
func floatToIntBits(value float32) int32 {
	result := math.Float32bits(value)
	if (result&f_EXP_BIT_MASK == f_EXP_BIT_MASK) && (result&f_SIGNIF_BIT_MASK != 0) {
		result = 0x7fc00000
	}
	return int32(result)
}

// doubleToLongBits实现Java中的doubleToLongBits
func doubleToLongBits(value float64) int64 {
	result := math.Float64bits(value)
	if (result&d_EXP_BIT_MASK == d_EXP_BIT_MASK) && (result&d_SIGNIF_BIT_MASK != 0) {
		result = 0x7ff8000000000000
	}
	return int64(result)
}

// JFloat Java float的兼容类型
type JFloat float32

func (self JFloat) HashCode() int32 {
	return floatToIntBits(float32(self))
}

// JDouble Java double的兼容类型
type JDouble float64

func (self JDouble) HashCode() int32 {
	return JLong(doubleToLongBits(float64(self))).HashCode()
}
