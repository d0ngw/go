package common

import (
	gjson "encoding/json"
	"fmt"
	"testing"
)

type TestJSON struct {
	ID   int64                  `json:"id"`
	Data map[string]interface{} `json:"data"`
	Fid  float64                `json:"fid"`
}

func TestJSONUnmarshal(t *testing.T) {
	var (
		packet = []byte(`{"id":102410241023,"fid":1.2,"data":{"b":12356332453}}`)
		err    error
	)

	fmt.Println("Use json")
	m := map[string]interface{}{}
	err = gjson.Unmarshal(packet, &m)
	fmt.Println("err:", err, "m:", m)
	mjsonStr, err := gjson.Marshal(m)
	fmt.Println("err:", err, "mjsonStr:", string(mjsonStr))

	fmt.Println("Use json useNumber")
	m2 := map[string]interface{}{}
	err = UnmarshalUseNumber(packet, &m2)
	fmt.Println("err:", err, "m2:", m2)
	mjsonStr, err = gjson.Marshal(m2)
	fmt.Println("err:", err, "mjsonStr:", string(mjsonStr))

	fmt.Println("tj")
	tj := &TestJSON{}

	err = gjson.Unmarshal(packet, tj)
	fmt.Println("err:", err, "use default unmarshal tj:", tj)
	mjsonStr, err = gjson.Marshal(tj)
	fmt.Println("err:", err, "mjsonStr:", string(mjsonStr))

	err = UnmarshalUseNumber(packet, tj)
	fmt.Println("err:", err, "use usenumber unmarshal tj:", tj)
}
