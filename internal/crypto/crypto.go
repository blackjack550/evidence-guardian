package crypto

import "fmt"

type Method string

const (
	MethodDPAPI     Method = "dpapi"
	MethodPassphrase Method = "passphrase"
)

type Config struct {
	Method   Method
	Passphrase string
}

func Encrypt(plaintext []byte, cfg Config) ([]byte, error) {
	switch cfg.Method {
	case MethodPassphrase:
		if cfg.Passphrase == "" {
			return nil, fmt.Errorf("口令加密需要设置密码")
		}
		return EncryptWithPassphrase(plaintext, cfg.Passphrase)
	default:
		return Protect(plaintext)
	}
}

func Decrypt(ciphertext []byte, cfg Config) ([]byte, error) {
	switch cfg.Method {
	case MethodPassphrase:
		if cfg.Passphrase == "" {
			return nil, fmt.Errorf("口令解密需要设置密码")
		}
		return DecryptWithPassphrase(ciphertext, cfg.Passphrase)
	default:
		return Unprotect(ciphertext)
	}
}
