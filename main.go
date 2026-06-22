package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"

	"evidence-guardian/internal/config"
	"evidence-guardian/internal/crypto"
	"evidence-guardian/internal/storage"
	"evidence-guardian/internal/trigger"
	"evidence-guardian/ui/server"
	"evidence-guardian/ui/tray"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("═══════════════════════════════════════════")
			log.Printf("程序崩溃! panic: %v", r)
			log.Printf("堆栈: %s", debug.Stack())
			log.Printf("═══════════════════════════════════════════")
		}
	}()

	decryptCmd := flag.String("decrypt", "", "解密证据文件: -decrypt=口令 -dir=证据目录")
	decryptDir := flag.String("dir", "./evidence", "待解密的证据目录")
	flag.Parse()

	if *decryptCmd != "" {
		runDecrypt(*decryptCmd, *decryptDir)
		return
	}

	exeDir := filepath.Dir(os.Args[0])
	cfgPath := filepath.Join(exeDir, "config.yaml")

	// Open log file (append mode)
	logPath := filepath.Join(exeDir, "evidence-guardian.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(io.MultiWriter(logFile, os.Stderr))
		defer logFile.Close()
	} else {
		log.SetOutput(os.Stderr)
	}

	// Open crash log (separate file, always create new)
	crashPath := filepath.Join(exeDir, "crash.log")
	crashFile, err := os.OpenFile(crashPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer crashFile.Close()
	}

	logCrash := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		log.Print(msg)
		if crashFile != nil {
			fmt.Fprintln(crashFile, msg)
			crashFile.Sync()
		}
	}

	logCrash("=== 证据卫士 v0.2.0 启动 ===")
	logCrash("工作目录: %s", exeDir)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		logCrash("加载配置失败: %v", err)
		return
	}

	if !filepath.IsAbs(cfg.Storage.Path) {
		cfg.Storage.Path = filepath.Join(exeDir, cfg.Storage.Path)
	}

	store, err := storage.New(cfg.Storage)
	if err != nil {
		logCrash("初始化存储失败: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := trigger.NewEngine(cfg, store)
	engine.SetCrashLog(func(msg string) {
		logCrash(msg)
	})

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

	// Monitor for exit
	exitCh := make(chan string, 1)
	go func() {
		tray.Run(cfg, engine, store, webServer)
		exitCh <- "tray_exit"
	}()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		exitCh <- fmt.Sprintf("signal_%v", sig)
	}()

	reason := <-exitCh
	logCrash("=== 程序退出 原因:%s ===", reason)

	engine.Stop()
	webServer.Stop()
	store.Close()
}

func runDecrypt(passphrase, dir string) {
	fmt.Println("证据卫士 — 批量解密工具")
	fmt.Println(strings.Repeat("-", 40))

	method := crypto.MethodDPAPI
	if passphrase != "" {
		method = crypto.MethodPassphrase
	}
	cc := crypto.Config{Method: method, Passphrase: passphrase}

	var total, success int
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".enc") {
			return nil
		}

		total++
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("❌ 读取失败: %s (%v)\n", path, err)
			return nil
		}

		plain, err := crypto.Decrypt(data, cc)
		if err != nil {
			fmt.Printf("❌ 解密失败: %s (%v)\n", path, err)
			return nil
		}

		outPath := strings.TrimSuffix(path, ".enc")
		if err := os.WriteFile(outPath, plain, 0644); err != nil {
			fmt.Printf("❌ 写入失败: %s (%v)\n", outPath, err)
			return nil
		}

		fmt.Printf("✅ 解密成功: %s (%d KB)\n", outPath, len(plain)/1024)
		success++
		return nil
	})

	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("总计: %d 个文件, 成功: %d 个\n", total, success)
	if success > 0 {
		fmt.Printf("明文文件已保存在原目录，文件名已去除 .enc 后缀\n")
	}
}
