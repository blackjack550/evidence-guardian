package trigger

import (
	"context"
	"log"
	"time"

	"evidence-guardian/internal/capture"
	"evidence-guardian/internal/config"
	"evidence-guardian/internal/ocr"
	"evidence-guardian/internal/storage"
)

type Engine struct {
	cfg      *config.Config
	store    *storage.Manager
	titleMon *TitleMonitor
	hotkey   *HotkeyManager
}

func NewEngine(cfg *config.Config, store *storage.Manager) *Engine {
	return &Engine{
		cfg:      cfg,
		store:    store,
		titleMon: NewTitleMonitor(cfg.Keywords, 2),
	}
}

func (e *Engine) Start(ctx context.Context) {
	go e.titleLoop(ctx)

	if e.cfg.OCR.Enabled {
		go e.ocrLoop(ctx)
	}

	go e.hotkeyLoop(ctx)

	log.Println("触发引擎已启动")
}

func (e *Engine) titleLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			matches, err := e.titleMon.Scan()
			if err != nil {
				continue
			}
			for _, m := range matches {
				log.Printf("[标题触发] 关键词:%s 窗口:%s\n", m.Keyword, m.Title)
				e.OnTrigger("title", m.Keyword, m.Window)
			}
		}
	}
}

func (e *Engine) ocrLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(e.cfg.OCR.IntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !e.cfg.OCR.Enabled {
				continue
			}
			go e.doOCRCheck()
		}
	}
}

func (e *Engine) doOCRCheck() {
	var targets []capture.AppTarget
	for _, t := range e.cfg.Targets {
		if t.Enabled {
			targets = append(targets, capture.AppTarget{
				Name: t.Name, Process: t.Process,
				WindowClass: t.WindowClass, Enabled: t.Enabled,
			})
		}
	}

	windows, err := capture.FindTargetWindows(targets)
	if err != nil || len(windows) == 0 {
		return
	}

	for _, w := range windows {
		// TODO: capture window image -> OCR -> match keywords
		_ = w
	}
}

func (e *Engine) hotkeyLoop(ctx context.Context) {
	hm, err := NewHotkeyManager()
	if err != nil {
		log.Printf("初始化热键失败: %v\n", err)
		return
	}
	e.hotkey = hm

	hm.Register(e.cfg.Hotkey.Modifiers, e.cfg.Hotkey.KeyCode, func() {
		log.Println("[用户手动触发] 热键取证")
		cfg := e.cfg
		_ = cfg
	})

	go hm.Start()
	<-ctx.Done()
	hm.Stop()
}

func (e *Engine) OnTrigger(source string, keyword string, win capture.WindowInfo) {
	record := &storage.EvidenceRecord{
		TriggerSource: source,
		Keyword:       keyword,
		WindowTitle:   win.Title,
		WindowClass:   win.ClassName,
		ProcessID:     win.ProcessID,
		Rect:          win.Rect,
	}

	e.store.SaveRecord(record)
	log.Printf("[取证] %s 触发证据已保存\n", keyword)
}

func (e *Engine) Stop() {
	if e.hotkey != nil {
		e.hotkey.Stop()
	}
}
