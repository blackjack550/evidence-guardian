package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltLen  = 16
	nonceLen = 12
	keyLen   = 32
	iter     = 600000
)

func EncryptWithPassphrase(plaintext []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("生成盐值失败: %w", err)
	}

	key := pbkdf2.Key([]byte(passphrase), salt, iter, keyLen, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES失败: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成随机数失败: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM失败: %w", err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	// Format: salt(16) + nonce(12) + ciphertext+tag
	result := make([]byte, 0, saltLen+nonceLen+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

func DecryptWithPassphrase(ciphertext []byte, passphrase string) ([]byte, error) {
	if len(ciphertext) < saltLen+nonceLen {
		return nil, fmt.Errorf("数据太短")
	}

	salt := ciphertext[:saltLen]
	nonce := ciphertext[saltLen : saltLen+nonceLen]
	encData := ciphertext[saltLen+nonceLen:]

	key := pbkdf2.Key([]byte(passphrase), salt, iter, keyLen, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES失败: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM失败: %w", err)
	}

	plaintext, err := aesgcm.Open(nil, nonce, encData, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败（口令错误或数据损坏）: %w", err)
	}

	return plaintext, nil
}
