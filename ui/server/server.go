package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"

	"evidence-guardian/internal/config"
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

	// API
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/trigger", s.handleTrigger)
	mux.HandleFunc("/api/status", s.handleStatus)

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


