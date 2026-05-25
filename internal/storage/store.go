package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"evidence-guardian/internal/capture"
	"evidence-guardian/internal/config"
	"evidence-guardian/internal/crypto"
)

type EvidenceRecord struct {
	ID            string
	Timestamp     time.Time
	TriggerSource string
	Keyword       string
	WindowTitle   string
	WindowClass   string
	ProcessID     uint32
	Rect          capture.RECT
	ScreenshotPath string
	VideoPath     string
	MetadataPath  string
}

type Manager struct {
	cfg     config.StorageConfig
	encrypt bool
}

func New(cfg config.StorageConfig) (*Manager, error) {
	if err := os.MkdirAll(cfg.Path, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}
	return &Manager{cfg: cfg, encrypt: cfg.Encrypt}, nil
}

func (m *Manager) SaveRecord(r *EvidenceRecord) error {
	dateStr := time.Now().Format("2006-01-02")
	timeStr := time.Now().Format("150405")
	dir := filepath.Join(m.cfg.Path, dateStr, fmt.Sprintf("%s_%s", timeStr, r.Keyword))

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建证据目录失败: %w", err)
	}

	metaPath := filepath.Join(dir, "metadata.json")
	screenshotPath := filepath.Join(dir, fmt.Sprintf("%s_screenshot.png", r.WindowClass))
	videoPath := filepath.Join(dir, fmt.Sprintf("%s_video.mp4", r.WindowClass))

	r.Timestamp = time.Now()
	r.MetadataPath = metaPath
	r.ScreenshotPath = screenshotPath
	r.VideoPath = videoPath

	if m.encrypt {
		r.ScreenshotPath += ".enc"
		r.VideoPath += ".enc"
		r.MetadataPath += ".enc"
	}

	return nil
}

func (m *Manager) WriteEncryptedFile(path string, data []byte) error {
	if m.encrypt {
		var err error
		data, err = crypto.Protect(data)
		if err != nil {
			return fmt.Errorf("加密失败: %w", err)
		}
	}
	return os.WriteFile(path, data, 0600)
}

func (m *Manager) ReadEncryptedFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if filepath.Ext(path) == ".enc" {
		return crypto.Unprotect(data)
	}
	return data, nil
}

func (m *Manager) CheckQuota() error {
	if !m.cfg.AutoClean {
		return nil
	}
	// TODO: calculate directory size and clean old records
	return nil
}

func (m *Manager) Close() error {
	return nil
}
