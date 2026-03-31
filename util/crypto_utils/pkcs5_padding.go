package crypto_utils

import "bytes"

// PKCS5Padding 填充明文
func PKCS5Padding(plaintext []byte, blockSize int) []byte {
	padding := blockSize - len(plaintext)%blockSize
	paddedBytes := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(plaintext, paddedBytes...)
}

// PKCS5UnPadding 去除填充数据
func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	un := int(origData[length-1])
	return origData[:(length - un)]
}
