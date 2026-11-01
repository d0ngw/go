package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/subtle"
	"errors"
	"fmt"
)

// The format identifier is authenticated as additional data. Never fall back to
// unauthenticated CBC when a message fails version or authentication checks.
const aesEnvelope = "GTK\x01"

// PKCS5Padding pads data using PKCS#7 (historically named PKCS5 in this API).
func PKCS5Padding(data []byte, blockSize int) ([]byte, error) {
	if blockSize < 1 || blockSize > 255 {
		return nil, errors.New("invalid padding block size")
	}
	padding := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...), nil
}

// PKCS5UnPadding removes PKCS padding after validating every padding byte.
// It does not authenticate the input. AES-CBC decryption also validates block size.
func PKCS5UnPadding(data []byte) ([]byte, error) {
	return unpad(data, 255)
}

func unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("invalid padding")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, errors.New("invalid padding")
	}
	valid := 1
	for _, b := range data[len(data)-padding:] {
		valid &= subtle.ConstantTimeByteEq(b, byte(padding))
	}
	if valid != 1 {
		return nil, errors.New("invalid padding")
	}
	return data[:len(data)-padding], nil
}

// AesEncrypt encrypts data using versioned AES-GCM with a fresh random nonce.
// The output contains the format identifier, nonce, ciphertext, and tag.
// A key must encrypt fewer than 2^32 messages before it is rotated.
func AesEncrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, err
	}
	return aead.Seal([]byte(aesEnvelope), nil, data, []byte(aesEnvelope)), nil
}

// AesDecrypt authenticates and decrypts data produced by AesEncrypt.
// Legacy ciphertext must be migrated explicitly using AesDecryptLegacy.
func AesDecrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(data, []byte(aesEnvelope)) {
		return nil, errors.New("unsupported AES ciphertext format")
	}
	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, nil, data[len(aesEnvelope):], []byte(aesEnvelope))
	if err != nil {
		return nil, errors.New("AES ciphertext authentication failed")
	}
	return plaintext, nil
}

// AesEncryptWithIV encrypts legacy AES-CBC data using an explicit 16-byte IV.
// Deprecated: use AesEncrypt, which generates a nonce and authenticates its output.
// CBC requires an unpredictable IV and independent authentication by the caller.
func AesEncryptWithIV(data, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("invalid IV length %d", len(iv))
	}
	padded, err := PKCS5Padding(data, aes.BlockSize)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)
	return ciphertext, nil
}

// AesDecryptWithIV decrypts legacy AES-CBC data using an explicit 16-byte IV.
// Deprecated: migrate stored CBC data and use AesDecrypt for authenticated data.
func AesDecryptWithIV(data, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(iv) != aes.BlockSize {
		return nil, fmt.Errorf("invalid IV length %d", len(iv))
	}
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, errors.New("invalid AES-CBC ciphertext length")
	}
	plaintext := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, data)
	return unpad(plaintext, aes.BlockSize)
}

// AesDecryptLegacy decrypts ciphertext from the old AES-CBC helpers for migration.
// A nil or empty IV reproduces the old key-derived IV. CBC cannot authenticate
// ciphertext: call this only for trusted legacy data, then re-encrypt with AesEncrypt.
func AesDecryptLegacy(data, key, iv []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, aes.KeySizeError(len(key))
	}
	if len(iv) == 0 {
		iv = key[:aes.BlockSize]
	}
	return AesDecryptWithIV(data, key, iv)
}
