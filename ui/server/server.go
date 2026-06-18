package server

import (
	"archive/zip"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"evidence-guardian/internal/config"
	"evidence-guardian/internal/crypto"
	"evidence-guardian/internal/trigger"
	"gopkg.in/yaml.v3"
)

//go:embed templates
var templateFS embed.FS

type Server struct {
	cfg    *config.Config
	engine *trigger.Engine
	server *http.Server
	port   int
}

func New(cfg *config.Config, engine *trigger.Engine) *Server {
	return &Server{
		cfg:    cfg,
		engine: engine,
		port:   58080,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Static assets and pages
	tmpl := template.Must(template.New("").ParseFS(templateFS, "templates/*.html"))
	mux.HandleFunc("/", s.handlePage(tmpl, "index.html"))

	// Evidence page
	mux.HandleFunc("/evidence", s.handlePage(tmpl, "evidence.html"))

	// API
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/trigger", s.handleTrigger)
	mux.HandleFunc("/api/shortcut", s.handleShortcut)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/evidence", s.handleEvidenceList)
	mux.HandleFunc("/api/evidence/view", s.handleEvidenceView)
	mux.HandleFunc("/api/evidence/export", s.handleEvidenceExport)
	mux.HandleFunc("/api/evidence/export-all", s.handleEvidenceExportAll)

	s.server = &http.Server{
		Handler: mux,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	// Find available port
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// fallback to random port
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("启动管理服务器失败: %w", err)
		}
		s.port = listener.Addr().(*net.TCPAddr).Port
	} else {
		s.port = 58080
	}

	go s.server.Serve(listener)
	return nil
}

func (s *Server) Port() int { return s.port }

func (s *Server) Stop() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *Server) handlePage(tmpl *template.Template, name string) http.HandlerFunc {
	data := map[string]interface{}{
		"Port": s.port,
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.ExecuteTemplate(w, name, data)
	}
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Mask passphrase in API response for security
		masked := *s.cfg
		if masked.Storage.Passphrase != "" {
			masked.Storage.Passphrase = "********"
		}
		json.NewEncoder(w).Encode(masked)
	case http.MethodPost:
		var updated config.Config
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(updated.Keywords) > 0 {
			s.cfg.Keywords = updated.Keywords
		}
		if updated.DedupSec > 0 {
			s.cfg.DedupSec = updated.DedupSec
		}
		s.cfg.NotifyOnTrigger = updated.NotifyOnTrigger
		s.cfg.CaptureMode = updated.CaptureMode
		s.cfg.OCR.Enabled = updated.OCR.Enabled
		s.cfg.OCR.IntervalSec = updated.OCR.IntervalSec
		s.cfg.Capture.VideoDurationSec = updated.Capture.VideoDurationSec
		if updated.Storage.Path != "" {
			s.cfg.Storage.Path = updated.Storage.Path
		}
		if updated.Storage.MaxSizeGB > 0 {
			s.cfg.Storage.MaxSizeGB = updated.Storage.MaxSizeGB
		}
		s.cfg.Storage.Encrypt = updated.Storage.Encrypt
		s.cfg.Storage.EncryptMethod = updated.Storage.EncryptMethod
		if updated.Storage.Passphrase != "" {
			s.cfg.Storage.Passphrase = updated.Storage.Passphrase
		}
		if updated.Hotkey.Modifiers != 0 {
			s.cfg.Hotkey.Modifiers = updated.Hotkey.Modifiers
		}
		if updated.Hotkey.KeyCode != 0 {
			s.cfg.Hotkey.KeyCode = updated.Hotkey.KeyCode
		}
		s.saveConfig()
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) saveConfig() {
	data, err := yaml.Marshal(s.cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "saveConfig marshal error: %v\n", err)
		return
	}
	exeDir := filepath.Dir(os.Args[0])
	cfgPath := filepath.Join(exeDir, "config.yaml")
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "saveConfig write error: %v\n", err)
	}
}

func (s *Server) handleShortcut(w http.ResponseWriter, r *http.Request) {
	browser := r.URL.Query().Get("browser")
	if browser != "chrome" && browser != "edge" {
		json.NewEncoder(w).Encode(map[string]string{"msg": "不支持"})
		return
	}

	var exePath, shortcutName string
	if browser == "chrome" {
		exePath = "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
		shortcutName = "Chrome（证据卫士调试模式）"
	} else {
		exePath = "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"
		shortcutName = "Edge（证据卫士调试模式）"
	}

	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		json.NewEncoder(w).Encode(map[string]string{"msg": "未找到" + browser + "，请在浏览器设置中确认安装路径"})
		return
	}

	desktop := os.Getenv("USERPROFILE") + "\\Desktop"
	ps := fmt.Sprintf(`
$ws = New-Object -ComObject WScript.Shell
$s = $ws.CreateShortcut('%s\\%s.lnk')
$s.TargetPath = '%s'
$s.Arguments = '--remote-debugging-port=9222'
$s.Save()
`, desktop, shortcutName, exePath)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Run()

	json.NewEncoder(w).Encode(map[string]string{"msg": fmt.Sprintf("已生成到桌面：%s", shortcutName)})
}

func (s *Server) handleTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.engine.ManualTrigger("web_panel")
	json.NewEncoder(w).Encode(map[string]string{"status": "triggered"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	exeDir := filepath.Dir(os.Args[0])

	tesseractOk := false
	if _, err := exec.LookPath("tesseract"); err == nil {
		tesseractOk = true
	} else {
		for _, p := range []string{
			filepath.Join(exeDir, "tesseract", "tesseract.exe"),
			"C:\\Program Files\\Tesseract-OCR\\tesseract.exe",
			"C:\\Program Files (x86)\\Tesseract-OCR\\tesseract.exe",
		} {
			if _, err := os.Stat(p); err == nil {
				tesseractOk = true
				break
			}
		}
	}

	ffmpegOk := false
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		ffmpegOk = true
	} else {
		for _, p := range []string{
			filepath.Join(exeDir, "ffmpeg.exe"),
			"C:\\ffmpeg\\bin\\ffmpeg.exe",
			"C:\\Program Files\\ffmpeg\\bin\\ffmpeg.exe",
		} {
			if _, err := os.Stat(p); err == nil {
				ffmpegOk = true
				break
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"capture_mode":   s.cfg.CaptureMode,
		"notify_mode":    s.cfg.NotifyOnTrigger,
		"ocr_enabled":    s.cfg.OCR.Enabled,
		"ocr_interval":   s.cfg.OCR.IntervalSec,
		"dedup_sec":      s.cfg.DedupSec,
		"tesseract":      tesseractOk,
		"ffmpeg":         ffmpegOk,
		"targets_count":  len(s.cfg.Targets),
	})
}

func (s *Server) handleEvidenceList(w http.ResponseWriter, r *http.Request) {
	root := s.cfg.Storage.Path
	var files []map[string]interface{}

	filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".enc") && !strings.HasSuffix(path, ".png") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = strings.ReplaceAll(rel, "\\", "/")
		dateDir := filepath.Dir(rel)
		name := info.Name()
		isEnc := strings.HasSuffix(name, ".enc")
		displayName := strings.TrimSuffix(name, ".enc")
		if strings.HasSuffix(displayName, ".png") {
			displayName = strings.TrimSuffix(displayName, ".png") + " [截图]"
		}
		files = append(files, map[string]interface{}{
			"path":     rel,
			"name":     displayName,
			"date":     dateDir,
			"size":     info.Size(),
			"encrypted": isEnc,
		})
		return nil
	})

	sort.Slice(files, func(i, j int) bool {
		return files[i]["date"].(string) > files[j]["date"].(string)
	})

	enc := json.NewEncoder(w)
	if len(files) == 0 {
		enc.Encode([]map[string]interface{}{})
		return
	}
	enc.Encode(files)
}

func (s *Server) handleEvidenceView(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	if rel == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}
	rel = strings.ReplaceAll(rel, "/", "\\")
	fullPath := filepath.Join(s.cfg.Storage.Path, rel)

	// Security: prevent directory traversal
	absRoot, _ := filepath.Abs(s.cfg.Storage.Path)
	absPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absPath, absRoot) {
		http.Error(w, "invalid path", http.StatusForbidden)
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, "文件不存在", http.StatusNotFound)
		return
	}

	if strings.HasSuffix(fullPath, ".enc") {
		method := crypto.Method(s.cfg.Storage.EncryptMethod)
		cc := crypto.Config{Method: method, Passphrase: s.cfg.Storage.Passphrase}
		data, err = crypto.Decrypt(data, cc)
		if err != nil {
			http.Error(w, "解密失败", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(data)
}

func (s *Server) handleEvidenceExport(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	if rel == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}
	rel = strings.ReplaceAll(rel, "/", "\\")
	fullPath := filepath.Join(s.cfg.Storage.Path, rel)

	absRoot, _ := filepath.Abs(s.cfg.Storage.Path)
	absPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absPath, absRoot) {
		http.Error(w, "invalid path", http.StatusForbidden)
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, "文件不存在", http.StatusNotFound)
		return
	}

	isEnc := strings.HasSuffix(fullPath, ".enc")
	if isEnc {
		method := crypto.Method(s.cfg.Storage.EncryptMethod)
		cc := crypto.Config{Method: method, Passphrase: s.cfg.Storage.Passphrase}
		data, err = crypto.Decrypt(data, cc)
		if err != nil {
			http.Error(w, "解密失败", http.StatusInternalServerError)
			return
		}
	}

	// Return decrypted file as download
	fileName := filepath.Base(rel)
	dlName := strings.TrimSuffix(fileName, ".enc")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, dlName))
	w.Write(data)
}

func (s *Server) handleEvidenceExportAll(w http.ResponseWriter, r *http.Request) {
	root := s.cfg.Storage.Path
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="evidence_export_%s.zip"`, time.Now().Format("20060102_150405")))

	zw := zip.NewWriter(w)
	defer zw.Close()

	cc := crypto.Config{
		Method:     crypto.Method(s.cfg.Storage.EncryptMethod),
		Passphrase: s.cfg.Storage.Passphrase,
	}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		rel = strings.ReplaceAll(rel, "\\", "/")

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		isEnc := strings.HasSuffix(path, ".enc")
		if isEnc {
			data, err = crypto.Decrypt(data, cc)
			if err != nil {
				return nil
			}
			rel = strings.TrimSuffix(rel, ".enc")
		}

		f, _ := zw.Create(rel)
		if f != nil {
			f.Write(data)
		}
		return nil
	})
}


