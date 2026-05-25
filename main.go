package main

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"evidence-guardian/internal/config"
	"evidence-guardian/internal/storage"
	"evidence-guardian/internal/trigger"
	"evidence-guardian/ui/server"
	"evidence-guardian/ui/tray"
)

func main() {
	logFile, err := os.OpenFile(
		filepath.Join(filepath.Dir(os.Args[0]), "evidence-guardian.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644,
	)
	if err == nil {
		log.SetOutput(io.MultiWriter(logFile, os.Stderr))
		defer logFile.Close()
	} else {
		log.SetOutput(os.Stderr)
	}

	log.Println("证据卫士 v0.1.0 — 劳动者权益保护取证系统")

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	store, err := storage.New(cfg.Storage)
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := trigger.NewEngine(cfg, store)

	webServer := server.New(cfg, engine)
	if err := webServer.Start(ctx); err != nil {
		log.Printf("启动管理面板失败: %v", err)
	}

	engine.SetNotifyHandler(func(title, message string) {
		tray.ShowNotify(title, message)
	})

	engine.Start(ctx)

	if webServer.Port() > 0 {
		log.Printf("管理面板: http://127.0.0.1:%d", webServer.Port())
	}

	tray.Run(cfg, engine, store, webServer)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	engine.Stop()
	webServer.Stop()
	store.Close()
}
