package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Keywords        []string       `json:"keywords" yaml:"keywords"`
	Targets         []TargetApp    `json:"targets" yaml:"targets"`
	OCR             OCRConfig      `json:"ocr" yaml:"ocr"`
	Capture         CaptureConfig  `json:"capture" yaml:"capture"`
	Storage         StorageConfig  `json:"storage" yaml:"storage"`
	Hotkey          HotkeyConfig   `json:"hotkey" yaml:"hotkey"`
	AutoStart       bool           `json:"auto_start" yaml:"auto_start"`
	CaptureMode     string         `json:"capture_mode" yaml:"capture_mode"`
	NotifyOnTrigger string         `json:"notify_on_trigger" yaml:"notify_on_trigger"`
}

type TargetApp struct {
	Name        string `json:"name" yaml:"name"`
	Process     string `json:"process" yaml:"process"`
	WindowClass string `json:"window_class" yaml:"window_class"`
	Enabled     bool   `json:"enabled" yaml:"enabled"`
}

type OCRConfig struct {
	IntervalSec int  `json:"interval_sec" yaml:"interval_sec"`
	Enabled     bool `json:"enabled" yaml:"enabled"`
}

type CaptureConfig struct {
	VideoDurationSec  int `json:"video_duration_sec" yaml:"video_duration_sec"`
	VideoFPS          int `json:"video_fps" yaml:"video_fps"`
	ScreenshotQuality int `json:"screenshot_quality" yaml:"screenshot_quality"`
}

type StorageConfig struct {
	Path       string `json:"path" yaml:"path"`
	MaxSizeGB  int    `json:"max_size_gb" yaml:"max_size_gb"`
	AutoClean  bool   `json:"auto_clean" yaml:"auto_clean"`
	Encrypt    bool   `json:"encrypt" yaml:"encrypt"`
	EncryptMethod string `json:"encrypt_method" yaml:"encrypt_method"` // dpapi | passphrase
	Passphrase string `json:"passphrase" yaml:"passphrase"`
}

type HotkeyConfig struct {
	Capture   string `json:"capture" yaml:"capture"`
	Modifiers int    `json:"modifiers" yaml:"modifiers"`
	KeyCode   int    `json:"key_code" yaml:"key_code"`
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
