package trigger

import (
	"testing"
)

func TestTitleMonitorKeywordMatch(t *testing.T) {
	mon := NewTitleMonitor([]string{"调薪", "降薪", "转岗"}, 2, 300)
	if mon == nil {
		t.Fatal("NewTitleMonitor 返回 nil")
	}
	if len(mon.keywords) != 3 {
		t.Errorf("关键词数量错误: got %d, want %d", len(mon.keywords), 3)
	}
}

func TestIsExcluded(t *testing.T) {
	mon := NewTitleMonitor([]string{"调薪"}, 2, 300, "C:\\evidence")
	tests := []struct {
		title string
		want  bool
	}{
		{"正常窗口标题", false},
		{"C:\\evidence - File Explorer", true},
		{"C:\\evidence\\2026-01-01\\151217_调薪 - 照片查看器", true},
		{"evidence 文件夹", false},
	}
	for _, tt := range tests {
		got := mon.isExcluded(tt.title)
		if got != tt.want {
			t.Errorf("isExcluded(%q) = %v, want %v", tt.title, got, tt.want)
		}
	}
}

func TestFingerprint(t *testing.T) {
	mon := NewTitleMonitor([]string{"调薪"}, 2, 300)
	hwnd := uintptr(12345)
	f1 := mon.fingerprint(hwnd, "调薪")
	f2 := mon.fingerprint(hwnd, "调薪")
	if f1 != f2 {
		t.Error("相同 HWND+关键词 的指纹应一致")
	}
	f3 := mon.fingerprint(hwnd, "降薪")
	if f1 == f3 {
		t.Error("不同关键词的指纹不应相同")
	}
	f4 := mon.fingerprint(uintptr(67890), "调薪")
	if f1 == f4 {
		t.Error("不同 HWND 的指纹不应相同")
	}
}

func TestExcludeProcesses(t *testing.T) {
	if !excludeProcesses["dllhost.exe"] {
		t.Error("应排除 dllhost.exe")
	}
	if !excludeProcesses["explorer.exe"] {
		t.Error("应排除 explorer.exe")
	}
	if excludeProcesses["chrome.exe"] {
		t.Error("不应排除 chrome.exe")
	}
}
