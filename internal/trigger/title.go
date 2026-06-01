package trigger

import (
	"crypto/md5"
	"fmt"
	"strings"
	"sync"
	"time"

	"evidence-guardian/internal/capture"
)

type TitleMonitor struct {
	keywords    []string
	interval    time.Duration
	mu          sync.Mutex
	recentKeys  map[string]time.Time
	cooldown    time.Duration
}

func NewTitleMonitor(keywords []string, intervalSec int) *TitleMonitor {
	return &TitleMonitor{
		keywords:   keywords,
		interval:   time.Duration(intervalSec) * time.Second,
		recentKeys: make(map[string]time.Time),
		cooldown:   60 * time.Second,
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
		if w.Title == "" {
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

func (m *TitleMonitor) fingerprint(hwnd uintptr, keyword string) string {
	h := md5.Sum([]byte(fmt.Sprintf("%d|%s", hwnd, keyword)))
	return string(h[:])
}
