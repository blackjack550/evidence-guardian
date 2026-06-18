package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig
	if len(cfg.Keywords) == 0 {
		t.Error("默认关键词不应为空")
	}
	if len(cfg.Targets) == 0 {
		t.Error("默认目标应用不应为空")
	}
	if cfg.Storage.Path == "" {
		t.Error("默认存储路径不应为空")
	}
	if cfg.Storage.MaxSizeGB <= 0 {
		t.Error("默认存储上限应大于0")
	}
	if cfg.OCR.IntervalSec <= 0 {
		t.Error("OCR间隔应大于0")
	}
	if cfg.Capture.VideoDurationSec <= 0 {
		t.Error("视频时长应大于0")
	}
}

func TestKeywordsNotEmpty(t *testing.T) {
	cfg := DefaultConfig
	found := false
	for _, kw := range cfg.Keywords {
		if kw == "调薪" {
			found = true
			break
		}
	}
	if !found {
		t.Error("默认关键词应包含「调薪」")
	}
}

func TestTargetsHaveRequiredApps(t *testing.T) {
	cfg := DefaultConfig
	required := []string{"Chrome", "Edge", "微信", "企业微信", "QQ", "钉钉", "飞书", "Outlook"}
	for _, name := range required {
		found := false
		for _, target := range cfg.Targets {
			if target.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("目标应用列表应包含「%s」", name)
		}
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "non_existent.yaml")
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载不存在的配置文件应返回默认配置: %v", err)
	}
	if cfg == nil {
		t.Fatal("应返回非nil配置")
	}
	// Verify the file was created
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("应自动生成配置文件")
	}
	// Cleanup
	os.Remove(cfgPath)
}

func TestLoadYAMLFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `
keywords:
  - 测试
targets:
  - name: TestApp
    process: test.exe
    window_class: TestClass
    enabled: true
storage:
  path: ./test_data
  max_size_gb: 10
  encrypt: true
capture_mode: desktop
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("写入测试配置文件失败: %v", err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("加载配置文件失败: %v", err)
	}
	if cfg.Keywords[0] != "测试" {
		t.Errorf("关键词解析错误: got %q, want %q", cfg.Keywords[0], "测试")
	}
	if cfg.Targets[0].Name != "TestApp" {
		t.Errorf("目标应用解析错误: got %q, want %q", cfg.Targets[0].Name, "TestApp")
	}
	if cfg.Storage.MaxSizeGB != 10 {
		t.Errorf("存储上限解析错误: got %d, want %d", cfg.Storage.MaxSizeGB, 10)
	}
	if cfg.CaptureMode != "desktop" {
		t.Errorf("采集模式解析错误: got %q, want %q", cfg.CaptureMode, "desktop")
	}
	os.Remove(cfgPath)
}
