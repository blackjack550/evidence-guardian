package trigger

import (
	"crypto/md5"
	"fmt"
	"strings"
	"sync"
	"time"

	"evidence-guardian/internal/capture"
)

var excludeProcesses = map[string]bool{
	"dllhost.exe":          true, // Windows Photo Viewer
	"Microsoft.Photos.exe": true, // Windows Photos app
	"PhotoViewer.dll":      true,
	"explorer.exe":         true, // File Explorer

}

type TitleMonitor struct {
	keywords    []string
	interval    time.Duration
	mu          sync.Mutex
	recentKeys  map[string]time.Time
	cooldown    time.Duration
	excludeDirs []string
}

func NewTitleMonitor(keywords []string, intervalSec int, dedupSec int, excludeDirs ...string) *TitleMonitor {
	cd := time.Duration(dedupSec) * time.Second
	if cd < 10*time.Second {
		cd = 300 * time.Second
	}
	return &TitleMonitor{
		keywords:    keywords,
		interval:    time.Duration(intervalSec) * time.Second,
		recentKeys:  make(map[string]time.Time),
		cooldown:    cd,
		excludeDirs: excludeDirs,
	}
}

type TitleMatch struct {
	Window  capture.WindowInfo
	Keyword string
	Title   string
}

func (m *TitleMonitor) Scan() ([]TitleMatch, error) {
	windows, err := capture.EnumWindows()
	if err != nil {
		return nil, err
	}

	var matches []TitleMatch
	m.mu.Lock()
	now := time.Now()
	for _, w := range windows {
		if w.Title == "" || excludeProcesses[strings.ToLower(w.ProcessName)] {
			continue
		}
		if m.isExcluded(w.Title) {
			continue
		}
		titleLower := strings.ToLower(w.Title)
		for _, kw := range m.keywords {
			if !strings.Contains(titleLower, strings.ToLower(kw)) {
				continue
			}
			key := m.fingerprint(w.HWND, kw)
			if last, ok := m.recentKeys[key]; ok && now.Sub(last) < m.cooldown {
				break // still in cooldown
			}
			m.recentKeys[key] = now
			matches = append(matches, TitleMatch{
				Window:  w,
				Keyword: kw,
				Title:   w.Title,
			})
			break
		}
	}
	// Clean old entries
	for k, v := range m.recentKeys {
		if now.Sub(v) > m.cooldown*2 {
			delete(m.recentKeys, k)
		}
	}
	m.mu.Unlock()
	return matches, nil
}

func (m *TitleMonitor) isExcluded(title string) bool {
	titleLower := strings.ToLower(title)
	for _, d := range m.excludeDirs {
		if strings.Contains(titleLower, strings.ToLower(d)) {
			return true
		}
	}
	return false
}

func (m *TitleMonitor) fingerprint(hwnd uintptr, keyword string) string {
	h := md5.Sum([]byte(fmt.Sprintf("%d|%s", hwnd, keyword)))
	return string(h[:])
}
