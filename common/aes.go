package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
)

// PKCS5Padding pkcs5 padding
func PKCS5Padding(ciphertext []byte, blockSize int) ([]byte, error) {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...), nil
}

// PKCS5UnPadding pkcs5 unpadding
func PKCS5UnPadding(origData []byte) ([]byte, error) {
	length := len(origData)
	if length == 0 {
		return nil, errors.New("Invalid length")
	}
	unpadding := int(origData[length-1])
	if unpadding < 0 {
		return nil, errors.New("Invalid unpadding")
	}
	end := length - unpadding
	if end < 0 || end > length {
		return nil, errors.New("Invalid end padding")
	}
	return origData[:(length - unpadding)], nil
}

// AesEncrypt 对data用key加密,使用PKCS5 Padding
func AesEncrypt(data, key []byte) (result []byte, err error) {
	defer func() {
		if reErr := recover(); reErr != nil {
			Errorf("AES Encrypt err:%s ", reErr)
			err = fmt.Errorf("AES Encrypt fail,%v", reErr)
		}
	}()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	data, err = PKCS5Padding(data, blockSize)
	if err != nil {
		return nil, err
	}
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(data))
	blockMode.CryptBlocks(crypted, data)
	return crypted, nil
}

// AesDecrypt 对data用key加密,使用PKCS5 Padding
func AesDecrypt(data, key []byte) (result []byte, err error) {
	defer func() {
		if reErr := recover(); reErr != nil {
			Errorf("AES Decrypt err:%s", reErr)
			err = fmt.Errorf("AES Decrypt fail,%v", reErr)
		}
	}()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(data))
	blockMode.CryptBlocks(origData, data)
	return PKCS5UnPadding(origData)
}
