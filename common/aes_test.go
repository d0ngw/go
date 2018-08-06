package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAES(t *testing.T) {
	s := "abc123"
	key := []byte("123456789abcdefghijklnm"[0:16])

	enc, err := AesEncrypt(nil, key)
	assert.Nil(t, err, "error:", err)
	dec, err := AesDecrypt(enc, key)
	assert.Nil(t, err, "error:", err)

	enc, err = AesEncrypt([]byte(s), key)
	assert.Nil(t, err, "error:", err)
	dec, err = AesDecrypt(enc, key)
	assert.EqualValues(t, []byte(s), dec)

	iv := []byte("1234567890123456")
	enc, err = AesEncryptWithIV([]byte(s), key, iv)
	assert.Nil(t, err, "error:", err)
	dec, err = AesDecryptWithIV(enc, key, iv)
	assert.Nil(t, err, "error:", err)
	assert.EqualValues(t, []byte(s), dec)
}
