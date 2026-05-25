package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"evidence-guardian/internal/config"
	"evidence-guardian/internal/storage"
	"evidence-guardian/internal/trigger"
	"evidence-guardian/ui/server"
	"evidence-guardian/ui/tray"
)

func main() {
	fmt.Println("证据卫士 v0.1.0 — 劳动者权益保护取证系统")

	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	store, err := storage.New(cfg.Storage)
	if err != nil {
		fmt.Printf("初始化存储失败: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := trigger.NewEngine(cfg, store)

	webServer := server.New(cfg, engine)
	if err := webServer.Start(ctx); err != nil {
		fmt.Printf("启动管理面板失败: %v\n", err)
	}

	engine.SetNotifyHandler(func(title, message string) {
		tray.ShowNotify(title, message)
	})

	engine.Start(ctx)

	tray.Run(cfg, engine, store, webServer)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	engine.Stop()
	webServer.Stop()
	store.Close()
}
