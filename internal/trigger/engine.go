package trigger

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"evidence-guardian/internal/capture"
	"evidence-guardian/internal/config"
	"evidence-guardian/internal/crypto"
	"evidence-guardian/internal/ocr"
	"evidence-guardian/internal/storage"
	"github.com/kbinani/screenshot"
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

	record := &storage.EvidenceRecord{TriggerSource: source, Keyword: "manual"}
	e.store.SaveRecord(record)
	shotDir := filepath.Dir(record.ScreenshotPath)

	img, err := capture.CaptureDesktop()
	if err != nil {
		log.Printf("截图失败: %v\n", err)
	} else {
		path, _ := capture.SavePNG(img, shotDir, "screenshot")
		path = e.maybeEncrypt(path)
		log.Printf("[取证] 截图: %s\n", path)
	}

	go func() {
		vidPath, err := capture.RecordDesktop(shotDir, "video",
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
	record := &storage.EvidenceRecord{
		TriggerSource: source,
		Keyword:       keyword,
		WindowTitle:   win.Title,
		WindowClass:   win.ClassName,
		ProcessID:     win.ProcessID,
		Rect:          win.Rect,
	}
	e.store.SaveRecord(record)
	shotDir := filepath.Dir(record.ScreenshotPath)

	img, err := capture.CaptureDesktop()
	if err == nil {
		path, _ := capture.SavePNG(img, shotDir, fmt.Sprintf("%s_%s", source, keyword))
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
	// Try once immediately
	e.doBrowserCheck(ctx)

	// Then retry every 30 seconds (not 5, to avoid flooding)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.doBrowserCheck(ctx)
		}
	}
}

func (e *Engine) doBrowserCheck(ctx context.Context) {
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

type ocrDedup struct {
	mu       sync.Mutex
	recent   map[string]time.Time
	cooldown time.Duration
}

func newOCRDedup() *ocrDedup {
	return &ocrDedup{recent: make(map[string]time.Time), cooldown: 30 * time.Second}
}

func (d *ocrDedup) allow(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	if last, ok := d.recent[key]; ok && now.Sub(last) < d.cooldown {
		return false
	}
	d.recent[key] = now
	for k, v := range d.recent {
		if now.Sub(v) > d.cooldown*2 {
			delete(d.recent, k)
		}
	}
	return true
}

var ocrOnce sync.Once

func (e *Engine) ocrLoop(ctx context.Context) {
	if e.ocrEngine == nil || !e.ocrEngine.IsReady() {
		ocrOnce.Do(func() { log.Println("OCR引擎未就绪，跳过IM内容检测") })
		return
	}

	interval := e.cfg.OCR.IntervalSec
	if interval < 3 {
		interval = 10
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	dedup := newOCRDedup()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.scanIMWindows(dedup)
		}
	}
}

func (e *Engine) scanIMWindows(dedup *ocrDedup) {
	// Find IM windows (all enabled targets except Chrome/Edge)
	var targets []capture.AppTarget
	for _, t := range e.cfg.Targets {
		if !t.Enabled {
			continue
		}
		if t.Process == "chrome.exe" || t.Process == "msedge.exe" {
			continue
		}
		targets = append(targets, capture.AppTarget{
			Name: t.Name, Process: t.Process,
			WindowClass: t.WindowClass, Enabled: true,
		})
	}

	windows, err := capture.FindTargetWindows(targets)
	if err != nil || len(windows) == 0 {
		return
	}

	for _, win := range windows {
		key := fmt.Sprintf("ocr_%d_%s", win.HWND, strings.Join(e.cfg.Keywords, ","))
		if !dedup.allow(key) {
			continue
		}

		// Capture the window region
		bounds := image.Rect(
			int(win.Rect.Left), int(win.Rect.Top),
			int(win.Rect.Right), int(win.Rect.Bottom),
		)
		if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
			continue
		}

		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			continue
		}

		// Save to temp and OCR
		tmpPath := filepath.Join(os.TempDir(), "ev_ocr_window.png")
		f, _ := os.Create(tmpPath)
		if f != nil {
			png.Encode(f, img)
			f.Close()
		}
		data, _ := os.ReadFile(tmpPath)
		os.Remove(tmpPath)

		text, err := e.ocrEngine.RecognizeBytes(data)
		if err != nil {
			continue
		}

		textLower := strings.ToLower(text)
		for _, kw := range e.cfg.Keywords {
			if strings.Contains(textLower, strings.ToLower(kw)) {
				log.Printf("[IM-OCR] %s 检测到关键词:%s", win.ClassName, kw)
				e.collect("ocr_im", kw, win)
				break
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
