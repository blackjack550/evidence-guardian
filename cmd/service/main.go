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
	go engine.Start(ctx)

	tray.Run(cfg, engine, store)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	engine.Stop()
	store.Close()
}
