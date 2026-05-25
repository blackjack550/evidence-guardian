package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"evidence-guardian/internal/config"
	"evidence-guardian/internal/crypto"
	"evidence-guardian/internal/trigger"
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
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/evidence", s.handleEvidenceList)
	mux.HandleFunc("/api/evidence/view", s.handleEvidenceView)

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
		json.NewEncoder(w).Encode(s.cfg)
	case http.MethodPost:
		var updated config.Config
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
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
		if updated.Hotkey.Modifiers != 0 {
			s.cfg.Hotkey.Modifiers = updated.Hotkey.Modifiers
		}
		if updated.Hotkey.KeyCode != 0 {
			s.cfg.Hotkey.KeyCode = updated.Hotkey.KeyCode
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"capture_mode":  s.cfg.CaptureMode,
		"notify_mode":   s.cfg.NotifyOnTrigger,
		"ocr_enabled":   s.cfg.OCR.Enabled,
		"targets_count": len(s.cfg.Targets),
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
		data, err = crypto.Unprotect(data)
		if err != nil {
			http.Error(w, "解密失败", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "image/png")
	w.Write(data)
}


