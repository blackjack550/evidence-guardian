package capture

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

type TabInfo struct {
	Title string
	URL   string
	Text  string
}

type BrowserDetector interface {
	GetTabs(ctx context.Context) ([]TabInfo, error)
	Name() string
}

func DetectBrowsers(ctx context.Context, keywords []string) []TabMatch {
	var matches []TabMatch

	for _, b := range []BrowserDetector{
		newChromeDetector("Chrome", "chrome.exe"),
		newChromeDetector("Edge", "msedge.exe"),
	} {
		tabs, err := b.GetTabs(ctx)
		if err != nil {
			log.Printf("[%s] 连接失败: %v", b.Name(), err)
			continue
		}
		log.Printf("[%s] 检测到 %d 个标签页", b.Name(), len(tabs))

		for _, tab := range tabs {
			for _, kw := range keywords {
				if containsAny(tab.Title, kw) || containsAny(tab.URL, kw) || containsAny(tab.Text, kw) {
					matches = append(matches, TabMatch{
						Browser: b.Name(),
						Keyword: kw,
						Tab:     tab,
					})
					break
				}
			}
		}
	}
	return matches
}

type TabMatch struct {
	Browser string
	Keyword string
	Tab     TabInfo
}

func containsAny(s, substr string) bool {
	// Simple byte-level comparison for ASCII keywords
	// For Unicode keywords, we compare directly (Go strings are UTF-8)
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func findFreePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

type portCheckResult struct {
	port int
	open bool
}

func checkPort(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
