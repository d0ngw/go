package common

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func ExampleJStringHashCode() {
	s := "𪚥"
	ss := []string{"", "1", "2", "中国", s, "abc中ddd过", "可以，你们商量好，我都支持"}
	for _, v := range ss {
		fmt.Println(JString(v).HashCode())
	}
	// Output:
	// 0
	// 49
	// 50
	// 642672
	// 1774428
	// -1825084178
	// 125810424
}

func ExampleJLongHashCode() {
	ll := []int64{0, 1, 2, 3, 4, 5, math.MaxInt64, math.MinInt64}
	for _, v := range ll {
		fmt.Println(JLong(v).HashCode())
	}
	// Output:
	// 0
	// 1
	// 2
	// 3
	// 4
	// 5
	// -2147483648
	// -2147483648
}

func ExampleJFloatHashCode() {
	ff := []float32{0, 1, 2, 3, 4, 5, math.MaxFloat32, math.SmallestNonzeroFloat32, float32(math.NaN())}
	for _, v := range ff {
		fmt.Println(JFloat(v).HashCode())
	}
	//Output:
	//0
	//1065353216
	//1073741824
	//1077936128
	//1082130432
	//1084227584
	//2139095039
	//1
	//2143289344
}

func ExampleJDoubleHashCode() {
	dd := []float64{0, 1, 2, 3, 4, 5, math.MaxFloat64, math.SmallestNonzeroFloat64, math.NaN()}
	for _, v := range dd {
		fmt.Println(JDouble(v).HashCode())
	}
	//Output:
	//0
	//1072693248
	//1073741824
	//1074266112
	//1074790400
	//1075052544
	//-2146435072
	//1
	//2146959360
}

func TestJCompatiable(t *testing.T) {
	s := "aa"
	st := reflect.TypeOf(s)
	var jt JTypeCompatible
	fmt.Println(reflect.TypeOf(jt), st)
}
