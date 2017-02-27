package cache

import (
	"errors"
	"reflect"

	"github.com/ugorji/go/codec"
)

var msgpackHandle = &codec.MsgpackHandle{}

func init() {
	msgpackHandle.MapType = reflect.TypeOf(map[string]interface{}(nil))
}

// MsgPackEncodeBytes encode data to bytes use msgpack
func MsgPackEncodeBytes(data interface{}) (bytes []byte, err error) {
	enc := codec.NewEncoderBytes(&bytes, msgpackHandle)
	err = enc.Encode(data)
	return
}

// MsgPackDecodeBytes decode bytes to dest use msgpack
func MsgPackDecodeBytes(bytes []byte, dest interface{}) (err error) {
	if len(bytes) == 0 {
		return errors.New("nil bytes to decode")
	}
	dec := codec.NewDecoderBytes(bytes, msgpackHandle)
	err = dec.Decode(dest)
	return
}
