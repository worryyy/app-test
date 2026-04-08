package encrypt

import (
	"crypto/des"
	"errors"
)

func DESECBEncrypt(plain, key []byte) ([]byte, error) {
	if len(key) < 8 {
		return nil, errors.New("des key length must be at least 8")
	}

	block, err := des.NewCipher(key[:8])
	if err != nil {
		return nil, err
	}

	data := pkcs7Pad(plain, block.BlockSize())
	out := make([]byte, len(data))
	for bs, be := 0, block.BlockSize(); bs < len(data); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Encrypt(out[bs:be], data[bs:be])
	}
	return out, nil
}

func DESECBDecrypt(cipherText, key []byte) ([]byte, error) {
	if len(key) < 8 {
		return nil, errors.New("des key length must be at least 8")
	}
	block, err := des.NewCipher(key[:8])
	if err != nil {
		return nil, err
	}
	if len(cipherText) == 0 || len(cipherText)%block.BlockSize() != 0 {
		return nil, errors.New("invalid des ciphertext length")
	}

	out := make([]byte, len(cipherText))
	for bs, be := 0, block.BlockSize(); bs < len(cipherText); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Decrypt(out[bs:be], cipherText[bs:be])
	}
	return pkcs7Unpad(out)
}
