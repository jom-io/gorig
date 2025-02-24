package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

func bytesRepeat(b byte, count int) []byte {
	result := make([]byte, count)
	for i := range result {
		result[i] = b
	}
	return result
}

func Encrypt(text, key string) (string, error) {
	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}

	pad := aes.BlockSize - len(text)%aes.BlockSize
	paddedText := append([]byte(text), bytesRepeat(byte(pad), pad)...)

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	ciphertext := make([]byte, len(paddedText))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedText)

	cipherWithIV := append(iv, ciphertext...)
	return base64.StdEncoding.EncodeToString(cipherWithIV), nil
}

func GenerateKey() string {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(key)
}

func Decrypt(encodedCipher, key string) (string, error) {
	k, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}

	cipherWithIV, err := base64.StdEncoding.DecodeString(encodedCipher)
	if err != nil {
		return "", err
	}

	if len(cipherWithIV) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	iv := cipherWithIV[:aes.BlockSize]
	ciphertext := cipherWithIV[aes.BlockSize:]

	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}

	plaintextPadded := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintextPadded, ciphertext)

	padding := int(plaintextPadded[len(plaintextPadded)-1])
	if padding > aes.BlockSize || padding == 0 {
		return "", fmt.Errorf("invalid padding")
	}
	for i := len(plaintextPadded) - padding; i < len(plaintextPadded); i++ {
		if int(plaintextPadded[i]) != padding {
			return "", fmt.Errorf("invalid padding")
		}
	}
	plaintext := plaintextPadded[:len(plaintextPadded)-padding]
	return string(plaintext), nil
}
