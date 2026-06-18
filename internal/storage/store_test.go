package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"evidence-guardian/internal/config"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.StorageConfig{
		Path:      tmpDir,
		MaxSizeGB: 10,
		AutoClean: true,
		Encrypt:   true,
	}
	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	if m == nil {
		t.Fatal("New 返回 nil")
	}
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("存储目录应被创建")
	}
}

func TestSaveRecord(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.StorageConfig{Path: tmpDir, MaxSizeGB: 10, Encrypt: true}
	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	r := &EvidenceRecord{
		TriggerSource: "test",
		Keyword:       "调薪",
	}
	if err := m.SaveRecord(r); err != nil {
		t.Fatalf("SaveRecord 失败: %v", err)
	}
	if r.ScreenshotPath == "" {
		t.Error("ScreenshotPath 不应为空")
	}
	if r.VideoPath == "" {
		t.Error("VideoPath 不应为空")
	}
	if r.MetadataPath == "" {
		t.Error("MetadataPath 不应为空")
	}
	// Check paths contain the keyword-based directory
	if !strings.Contains(r.ScreenshotPath, "调薪") {
		t.Errorf("ScreenshotPath 应包含关键词: %s", r.ScreenshotPath)
	}
	// Check encryption suffix
	if !strings.HasSuffix(r.ScreenshotPath, ".enc") {
		t.Errorf("加密模式下 ScreenshotPath 应有 .enc 后缀: %s", r.ScreenshotPath)
	}
}

func TestSaveRecordNoEncrypt(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.StorageConfig{Path: tmpDir, MaxSizeGB: 10, Encrypt: false}
	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	r := &EvidenceRecord{TriggerSource: "test", Keyword: "test"}
	m.SaveRecord(r)
	if strings.HasSuffix(r.ScreenshotPath, ".enc") {
		t.Error("非加密模式下不应有 .enc 后缀")
	}
}

func TestSaveRecordTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.StorageConfig{Path: tmpDir, MaxSizeGB: 10, Encrypt: false}
	m, _ := New(cfg)
	before := time.Now()
	r := &EvidenceRecord{TriggerSource: "test", Keyword: "test"}
	m.SaveRecord(r)
	after := time.Now()
	if r.Timestamp.Before(before) || r.Timestamp.After(after) {
		t.Error("Timestamp 应在合理范围内")
	}
}

func TestDirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.StorageConfig{Path: tmpDir, MaxSizeGB: 10, Encrypt: false}
	m, _ := New(cfg)
	r := &EvidenceRecord{TriggerSource: "manual", Keyword: "调薪"}
	m.SaveRecord(r)
	// Expected: tmpDir/2026-06-01/150000_调薪/screenshot.png
	dir := filepath.Dir(r.ScreenshotPath)
	parentDir := filepath.Base(filepath.Dir(dir))
	timeDir := filepath.Base(dir)
	// parentDir should be date (2026-06-01)
	if len(parentDir) != 10 || parentDir[4] != '-' || parentDir[7] != '-' {
		t.Errorf("父目录应为日期格式: %s", parentDir)
	}
	// timeDir should contain keyword
	if !strings.Contains(timeDir, "调薪") {
		t.Errorf("时间目录应包含关键词: %s", timeDir)
	}
	_ = parentDir
}

func TestWriteEncryptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.StorageConfig{Path: tmpDir, MaxSizeGB: 10, Encrypt: true}
	m, _ := New(cfg)
	// Use .enc extension so ReadEncryptedFile recognizes it as encrypted
	testPath := filepath.Join(tmpDir, "test.png.enc")
	err := m.WriteEncryptedFile(testPath, []byte("test data"))
	if err != nil {
		t.Fatalf("WriteEncryptedFile 失败: %v", err)
	}
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Error("加密文件应被创建")
	}
	data, err := m.ReadEncryptedFile(testPath)
	if err != nil {
		t.Fatalf("ReadEncryptedFile 失败: %v", err)
	}
	if string(data) != "test data" {
		t.Errorf("读取数据不符: got %q, want %q", string(data), "test data")
	}
	os.Remove(testPath)
}
