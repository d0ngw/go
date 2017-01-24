package common

import (
	"bytes"
	"encoding/json"
)

// 使用UserNumber进行解析,避免int64被错误地转为float64
func UnmarshalUseNumber(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.UseNumber()
	return dec.Decode(v)
}
