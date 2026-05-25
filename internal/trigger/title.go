package trigger

import (
	"strings"
	"time"

	"evidence-guardian/internal/capture"
)

type TitleMonitor struct {
	keywords []string
	interval time.Duration
}

func NewTitleMonitor(keywords []string, intervalSec int) *TitleMonitor {
	return &TitleMonitor{
		keywords: keywords,
		interval: time.Duration(intervalSec) * time.Second,
	}
}

type TitleMatch struct {
	Window   capture.WindowInfo
	Keyword  string
	Title    string
}

func (m *TitleMonitor) Scan() ([]TitleMatch, error) {
	windows, err := capture.EnumWindows()
	if err != nil {
		return nil, err
	}

	var matches []TitleMatch
	for _, w := range windows {
		if w.Title == "" {
			continue
		}
		titleLower := strings.ToLower(w.Title)
		for _, kw := range m.keywords {
			if strings.Contains(titleLower, strings.ToLower(kw)) {
				matches = append(matches, TitleMatch{
					Window:  w,
					Keyword: kw,
					Title:   w.Title,
				})
				break
			}
		}
	}
	return matches, nil
}
