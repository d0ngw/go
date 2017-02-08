package redis

import (
	"github.com/ugorji/go/codec"
)

var msgpackHandle codec.Handle = new(codec.MsgpackHandle)
