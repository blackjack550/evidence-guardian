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
	"evidence-guardian/internal/ocr"
	"evidence-guardian/internal/storage"
)

type Engine struct {
	cfg           *config.Config
	store         *storage.Manager
	titleMon      *TitleMonitor
	hotkey        *HotkeyManager
	ocrEngine     *ocr.Engine
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
	if e.cfg.CaptureMode != "browser" || e.cfg.OCR.Enabled {
		e.ocrEngine = ocr.New()
	}

	go e.titleLoop(ctx)
	go e.browserLoop(ctx)

	if e.ocrEngine != nil && e.ocrEngine.IsReady() {
		go e.ocrLoop(ctx)
	}

	go e.hotkeyLoop(ctx)
	log.Printf("触发引擎已启动 (模式:%s)", e.cfg.CaptureMode)
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
				e.collect("title", m.Keyword, m.Window)
			}
		}
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
	ts := time.Now().UnixMilli()

	img, err := capture.CaptureDesktop()
	if err != nil {
		log.Printf("截图失败: %v\n", err)
	} else {
		path, _ := capture.SavePNG(img, shotDir, fmt.Sprintf("manual_%s_%d", "screenshot", ts))
		path = e.maybeEncrypt(path)
		log.Printf("[取证] 截图: %s\n", path)
	}

	// Video recording (async, non-blocking)
	go func() {
		vidPath, err := capture.RecordDesktop(shotDir, fmt.Sprintf("manual_%d", ts),
			e.cfg.Capture.VideoDurationSec, e.cfg.Capture.VideoFPS)
		if err != nil {
			log.Printf("视频录制提示: %v\n", err)
			return
		}
		vidPath = e.maybeEncrypt(vidPath)
		log.Printf("[取证] 视频: %s\n", vidPath)
	}()

	e.notify("证据卫士", "截图已保存，视频录制中…")
}

func (e *Engine) collect(source string, keyword string, win capture.WindowInfo) {
	e.store.SaveRecord(&storage.EvidenceRecord{
		TriggerSource: source,
		Keyword:       keyword,
		WindowTitle:   win.Title,
		WindowClass:   win.ClassName,
		ProcessID:     win.ProcessID,
		Rect:          win.Rect,
	})

	shotDir := filepath.Join(e.cfg.Storage.Path, time.Now().Format("2006-01-02"))
	ts := time.Now().UnixMilli()

	img, err := capture.CaptureDesktop()
	if err == nil {
		path, _ := capture.SavePNG(img, shotDir, fmt.Sprintf("%s_%s_%d", source, keyword, ts))
		path = e.maybeEncrypt(path)
	}

	log.Printf("[取证] %s → %s\n", source, keyword)

	switch e.cfg.NotifyOnTrigger {
	case "toast":
		e.notify("证据卫士", fmt.Sprintf("检测到关键词「%s」，证据已采集", keyword))
	case "alert":
		e.notify("⚠️ 证据采集提醒",
			fmt.Sprintf("屏幕出现敏感关键词「%s」\n系统已自动采集证据", keyword))
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
				e.collect("browser_"+m.Browser, m.Keyword, win)
			}
		}
	}
}

func (e *Engine) maybeEncrypt(path string) string {
	if path == "" || !e.cfg.Storage.Encrypt {
		return path
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return path
	}
	encData, err := e.encryptData(data)
	if err != nil {
		return path
	}
	encPath := path + ".enc"
	os.WriteFile(encPath, encData, 0600)
	os.Remove(path)
	return encPath
}

func (e *Engine) encryptData(data []byte) ([]byte, error) {
	if e.cfg.Storage.EncryptMethod == "passphrase" && e.cfg.Storage.Passphrase != "" {
		return crypto.EncryptWithPassphrase(data, e.cfg.Storage.Passphrase)
	}
	return crypto.Protect(data)
}

func (e *Engine) notify(title, message string) {
	if e.notifyHandler != nil {
		e.notifyHandler(title, message)
	}
}

func (e *Engine) ocrLoop(ctx context.Context) {
	interval := e.cfg.OCR.IntervalSec
	if interval < 3 {
		interval = 10
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			matches, err := e.ocrEngine.ScanDesktop(e.cfg.Keywords)
			if err != nil {
				continue
			}
			for _, m := range matches {
				log.Printf("[OCR触发] 关键词:%s", m.Keyword)
				e.collect("ocr", m.Keyword, capture.WindowInfo{Title: m.Text})
			}
		}
	}
}

func (e *Engine) Stop() {
	if e.hotkey != nil {
		e.hotkey.Stop()
	}
	if e.ocrEngine != nil {
		e.ocrEngine.Close()
	}
}
