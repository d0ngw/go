package common

import (
	"bytes"
	"encoding/json"
)

// UnmarshalUseNumber preserves JSON numbers instead of converting them to float64.
func UnmarshalUseNumber(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.UseNumber()
	return dec.Decode(v)
}
