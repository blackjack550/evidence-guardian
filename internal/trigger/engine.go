package trigger

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"evidence-guardian/internal/capture"
	"evidence-guardian/internal/config"
	"evidence-guardian/internal/crypto"
	"evidence-guardian/internal/storage"
)

type Engine struct {
	cfg           *config.Config
	store         *storage.Manager
	titleMon      *TitleMonitor
	hotkey        *HotkeyManager
	notifyHandler func(title, message string)
}

func (e *Engine) SetNotifyHandler(h func(title, message string)) {
	e.notifyHandler = h
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

	go e.browserLoop(ctx)
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
		e.ManualTrigger("hotkey")
	})

	go hm.Start()
	<-ctx.Done()
	hm.Stop()
}

func (e *Engine) ManualTrigger(source string) {
	log.Printf("[手动取证] 来源:%s\n", source)
	e.notify("证据卫士", "正在采集当前屏幕证据…")

	shotDir := filepath.Join(e.cfg.Storage.Path, time.Now().Format("2006-01-02"))
	img, err := capture.CaptureDesktop()
	if err != nil {
		log.Printf("截图失败: %v\n", err)
		return
	}

	path, err := capture.SavePNG(img, shotDir, fmt.Sprintf("manual_%s", time.Now().Format("150405")))
	if err != nil {
		log.Printf("保存截图失败: %v\n", err)
		return
	}

	if e.cfg.Storage.Encrypt {
		encPath := path + ".enc"
		data, _ := os.ReadFile(path)
		encData, _ := crypto.Protect(data)
		os.WriteFile(encPath, encData, 0600)
		os.Remove(path)
		path = encPath
	}

	label := "截图"
	if e.cfg.Storage.Encrypt {
		label = "加密截图"
	}
	e.notify("证据卫士", fmt.Sprintf("%s已保存: %s", label, path))
	log.Printf("[取证] %s已保存: %s\n", label, path)
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

	shotDir := filepath.Join(e.cfg.Storage.Path, time.Now().Format("2006-01-02"))
	img, err := capture.CaptureDesktop()
	if err == nil {
		path, _ := capture.SavePNG(img, shotDir, fmt.Sprintf("%s_%s", source, keyword))
		if e.cfg.Storage.Encrypt && path != "" {
			encPath := path + ".enc"
			data, _ := os.ReadFile(path)
			encData, _ := crypto.Protect(data)
			os.WriteFile(encPath, encData, 0600)
			os.Remove(path)
		}
	}

	log.Printf("[取证] %s → %s\n", source, keyword)

	switch e.cfg.NotifyOnTrigger {
	case "toast":
		e.notify("证据卫士", fmt.Sprintf("检测到关键词「%s」，证据已采集", keyword))
	case "alert":
		e.notify("⚠️ 证据采集提醒",
			fmt.Sprintf("屏幕出现敏感关键词「%s」\n系统已自动采集证据，请勿操作异常关闭", keyword))
	}
}

func (e *Engine) notify(title, message string) {
	if e.notifyHandler != nil {
		e.notifyHandler(title, message)
	}
}

func (e *Engine) browserLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			matches := capture.DetectBrowsers(ctx, e.cfg.Keywords)
			for _, m := range matches {
				log.Printf("[浏览器触发] %s 关键词:%s URL:%s", m.Browser, m.Keyword, m.Tab.URL)
				win := capture.WindowInfo{Title: m.Tab.Title}
				if m.Tab.URL != "" {
					win.Title = m.Tab.URL + " - " + win.Title
				}
				e.OnTrigger("browser_"+m.Browser, m.Keyword, win)
			}
		}
	}
}

func (e *Engine) Stop() {
	if e.hotkey != nil {
		e.hotkey.Stop()
	}
}
