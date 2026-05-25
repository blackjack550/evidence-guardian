package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Keywords        []string       `yaml:"keywords"`
	Targets         []TargetApp    `yaml:"targets"`
	OCR             OCRConfig      `yaml:"ocr"`
	Capture         CaptureConfig  `yaml:"capture"`
	Storage         StorageConfig  `yaml:"storage"`
	Hotkey          HotkeyConfig   `yaml:"hotkey"`
	AutoStart       bool           `yaml:"auto_start"`
	CaptureMode     string         `yaml:"capture_mode"`     // browser | desktop | full
	NotifyOnTrigger string         `yaml:"notify_on_trigger"` // silent | toast | alert
}

type TargetApp struct {
	Name      string `yaml:"name"`
	Process   string `yaml:"process"`
	WindowClass string `yaml:"window_class"`
	Enabled   bool   `yaml:"enabled"`
}

type OCRConfig struct {
	IntervalSec int  `yaml:"interval_sec"`
	Enabled     bool `yaml:"enabled"`
}

type CaptureConfig struct {
	VideoDurationSec int  `yaml:"video_duration_sec"`
	VideoFPS         int  `yaml:"video_fps"`
	ScreenshotQuality int `yaml:"screenshot_quality"`
}

type StorageConfig struct {
	Path       string `yaml:"path"`
	MaxSizeGB  int    `yaml:"max_size_gb"`
	AutoClean  bool   `yaml:"auto_clean"`
	Encrypt    bool   `yaml:"encrypt"`
}

type HotkeyConfig struct {
	Capture   string `yaml:"capture"`
	Modifiers int    `yaml:"modifiers"`
	KeyCode   int    `yaml:"key_code"`
}

var DefaultConfig = Config{
	Keywords: []string{"调薪", "降薪", "转岗", "待岗", "辞退", "优化", "解除合同"},
	Targets: []TargetApp{
		{Name: "Chrome", Process: "chrome.exe", WindowClass: "Chrome_WidgetWin_1", Enabled: true},
		{Name: "Edge", Process: "msedge.exe", WindowClass: "Chrome_WidgetWin_1", Enabled: true},
		{Name: "Firefox", Process: "firefox.exe", WindowClass: "MozillaWindowClass", Enabled: true},
		{Name: "企业微信", Process: "WXWork.exe", WindowClass: "WXWorkMainWindow", Enabled: true},
		{Name: "钉钉", Process: "DingTalk.exe", WindowClass: "DingTalk", Enabled: true},
		{Name: "QQ", Process: "QQ.exe", WindowClass: "TXGuiFoundation", Enabled: true},
		{Name: "微信", Process: "WeChat.exe", WindowClass: "WeChatMainWndForPC", Enabled: true},
		{Name: "飞书", Process: "Feishu.exe", WindowClass: "Chrome_WidgetWin_0", Enabled: true},
	},
	OCR: OCRConfig{IntervalSec: 10, Enabled: true},
	Capture: CaptureConfig{VideoDurationSec: 8, VideoFPS: 10, ScreenshotQuality: 90},
	Storage: StorageConfig{Path: "./evidence", MaxSizeGB: 50, AutoClean: true, Encrypt: true},
	Hotkey:  HotkeyConfig{Modifiers: 6, KeyCode: 0x7B}, // Ctrl+Shift+F12
	AutoStart:       false,
	CaptureMode:     "browser",                          // browser | desktop | full
	NotifyOnTrigger: "silent",                           // silent | toast | alert
}

func Load(path string) (*Config, error) {
	cfg := DefaultConfig

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, saveDefault(path, &cfg)
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	return &cfg, nil
}

func saveDefault(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
