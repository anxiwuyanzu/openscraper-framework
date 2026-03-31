package crypto_utils

import (
	"crypto/aes"
	"crypto/cipher"
)

// CbcEncrypt AES cbc加密
func CbcEncrypt(origData, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//AES分组长度为128位，所以blockSize=16，单位字节
	blockSize := block.BlockSize()
	origData = PKCS5Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, iv) //初始向量的长度必须等于块block的长度16字节
	encrypted := make([]byte, len(origData))
	blockMode.CryptBlocks(encrypted, origData)
	return encrypted, nil
}

// CbcDecrypt AES cbc解密
func CbcDecrypt(encrypted, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//AES分组长度为128位，所以blockSize=16，单位字节
	blockMode := cipher.NewCBCDecrypter(block, iv) //初始向量的长度必须等于块block的长度16字节
	origData := make([]byte, len(encrypted))
	blockMode.CryptBlocks(origData, encrypted)
	origData = PKCS5UnPadding(origData)
	return origData, nil
}
