package crypto

import (
	"bytes"
	"testing"
)

func TestDPAPIProtectUnprotect(t *testing.T) {
	plain := []byte("测试证据数据 test evidence data 123!@#")
	enc, err := Protect(plain)
	if err != nil {
		t.Fatalf("Protect 失败: %v", err)
	}
	if bytes.Equal(enc, plain) {
		t.Error("加密后数据不应与原文相同")
	}
	if len(enc) == 0 {
		t.Error("加密后数据不应为空")
	}
	dec, err := Unprotect(enc)
	if err != nil {
		t.Fatalf("Unprotect 失败: %v", err)
	}
	if !bytes.Equal(dec, plain) {
		t.Errorf("解密结果与原文不符: got %v, want %v", dec, plain)
	}
}

func TestDPAPIEmptyInput(t *testing.T) {
	enc, err := Protect(nil)
	if err != nil {
		t.Fatalf("Protect nil 失败: %v", err)
	}
	if enc != nil {
		t.Error("nil 输入应返回 nil")
	}
	dec, err := Unprotect(nil)
	if err != nil {
		t.Fatalf("Unprotect nil 失败: %v", err)
	}
	if dec != nil {
		t.Error("nil 输入应返回 nil")
	}
	enc, err = Protect([]byte{})
	if err != nil {
		t.Fatalf("Protect empty 失败: %v", err)
	}
	if enc != nil {
		t.Error("空输入应返回 nil")
	}
}

func TestPassphraseEncryptDecrypt(t *testing.T) {
	plain := []byte("测试证据数据 test evidence data")
	passphrase := "MyTestPass123!@#"
	enc, err := EncryptWithPassphrase(plain, passphrase)
	if err != nil {
		t.Fatalf("EncryptWithPassphrase 失败: %v", err)
	}
	if bytes.Equal(enc, plain) {
		t.Error("加密后数据不应与原文相同")
	}
	// Test with correct passphrase
	dec, err := DecryptWithPassphrase(enc, passphrase)
	if err != nil {
		t.Fatalf("DecryptWithPassphrase 失败: %v", err)
	}
	if !bytes.Equal(dec, plain) {
		t.Errorf("解密结果与原文不符: got %v, want %v", dec, plain)
	}
	// Test with wrong passphrase
	_, err = DecryptWithPassphrase(enc, "wrong_password")
	if err == nil {
		t.Error("错误口令应返回错误")
	}
}

func TestPassphraseChineseChars(t *testing.T) {
	plain := []byte("调薪降薪转岗辞退优化解除合同")
	passphrase := "我的密码是中文的"
	enc, err := EncryptWithPassphrase(plain, passphrase)
	if err != nil {
		t.Fatalf("EncryptWithPassphrase 失败: %v", err)
	}
	dec, err := DecryptWithPassphrase(enc, passphrase)
	if err != nil {
		t.Fatalf("DecryptWithPassphrase 失败: %v", err)
	}
	if !bytes.Equal(dec, plain) {
		t.Errorf("中文解密结果不符: got %q, want %q", dec, plain)
	}
}

func TestPassphraseDifferentKeys(t *testing.T) {
	plain := []byte("same data")
	enc1, _ := EncryptWithPassphrase(plain, "pass1")
	enc2, _ := EncryptWithPassphrase(plain, "pass2")
	if bytes.Equal(enc1, enc2) {
		t.Error("不同口令加密结果不应相同")
	}
}

func TestUnifiedEncryptDecrypt(t *testing.T) {
	plain := []byte("unified test data")
	tests := []struct {
		name string
		cfg  Config
	}{
		{"DPAPI", Config{MethodDPAPI, ""}},
		{"Passphrase", Config{MethodPassphrase, "test123"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := Encrypt(plain, tt.cfg)
			if err != nil {
				t.Fatalf("Encrypt 失败: %v", err)
			}
			dec, err := Decrypt(enc, tt.cfg)
			if err != nil {
				t.Fatalf("Decrypt 失败: %v", err)
			}
			if !bytes.Equal(dec, plain) {
				t.Errorf("解密结果不符: got %v, want %v", dec, plain)
			}
		})
	}
}
