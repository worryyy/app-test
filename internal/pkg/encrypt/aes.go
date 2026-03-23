package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

func AESEncrypt(plainText, key string) (string, error) {
	keyBytes, err := normalizeAESKey([]byte(key))
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	data := pkcs7Pad([]byte(plainText), block.BlockSize())
	iv := keyBytes[:block.BlockSize()]
	encrypted := make([]byte, len(data))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, data)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func AESDecrypt(cipherText, key string) (string, error) {
	keyBytes, err := normalizeAESKey([]byte(key))
	if err != nil {
		return "", err
	}

	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 || len(raw)%block.BlockSize() != 0 {
		return "", errors.New("invalid aes ciphertext length")
	}

	iv := keyBytes[:block.BlockSize()]
	decrypted := make([]byte, len(raw))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, raw)
	decrypted, err = pkcs7Unpad(decrypted)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

func normalizeAESKey(key []byte) ([]byte, error) {
	switch {
	case len(key) <= 16:
		out := make([]byte, 16)
		copy(out, key)
		return out, nil
	case len(key) <= 24:
		out := make([]byte, 24)
		copy(out, key)
		return out, nil
	case len(key) <= 32:
		out := make([]byte, 32)
		copy(out, key)
		return out, nil
	default:
		return nil, errors.New("aes key too long")
	}
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	if padLen == 0 {
		padLen = blockSize
	}
	padding := make([]byte, padLen)
	for i := range padding {
		padding[i] = byte(padLen)
	}
	return append(data, padding...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("invalid padding")
	}
	padLen := int(data[len(data)-1])
	if padLen <= 0 || padLen > len(data) {
		return nil, errors.New("invalid padding")
	}
	for _, b := range data[len(data)-padLen:] {
		if int(b) != padLen {
			return nil, errors.New("invalid padding")
		}
	}
	return data[:len(data)-padLen], nil
}
