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
	encrypted := make([]byte, len(data))
	encryptECB(block, encrypted, data)
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

	decrypted := make([]byte, len(raw))
	decryptECB(block, decrypted, raw)
	decrypted, err = pkcs7Unpad(decrypted)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

func normalizeAESKey(key []byte) ([]byte, error) {
	switch len(key) {
	case 16, 24, 32:
		out := make([]byte, len(key))
		copy(out, key)
		return out, nil
	default:
		return nil, errors.New("invalid aes key length")
	}
}

func encryptECB(block cipher.Block, dst, src []byte) {
	blockSize := block.BlockSize()
	for start := 0; start < len(src); start += blockSize {
		block.Encrypt(dst[start:start+blockSize], src[start:start+blockSize])
	}
}

func decryptECB(block cipher.Block, dst, src []byte) {
	blockSize := block.BlockSize()
	for start := 0; start < len(src); start += blockSize {
		block.Decrypt(dst[start:start+blockSize], src[start:start+blockSize])
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
