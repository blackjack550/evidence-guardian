package crypto

import (
	"fmt"
	"syscall"
	"unsafe"
)

var procLocalFree = syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree")

type _DATA_BLOB struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

const (
	CRYPTPROTECT_UI_FORBIDDEN = 0x01
)

func Protect(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}

	var in _DATA_BLOB
	in.cbData = uint32(len(plaintext))
	in.pbData = &plaintext[0]

	var out _DATA_BLOB

	ret, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, 0, 0, 0,
		CRYPTPROTECT_UI_FORBIDDEN,
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("CryptProtectData 失败: %w", err)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))

	result := make([]byte, out.cbData)
	copy(result, unsafe.Slice(out.pbData, out.cbData))
	return result, nil
}

func Unprotect(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, nil
	}

	var in _DATA_BLOB
	in.cbData = uint32(len(ciphertext))
	in.pbData = &ciphertext[0]

	var out _DATA_BLOB

	ret, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0, 0, 0, 0,
		CRYPTPROTECT_UI_FORBIDDEN,
		uintptr(unsafe.Pointer(&out)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("CryptUnprotectData 失败: %w", err)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))

	result := make([]byte, out.cbData)
	copy(result, unsafe.Slice(out.pbData, out.cbData))
	return result, nil
}
