package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"evidence-guardian/internal/config"
	"evidence-guardian/internal/crypto"
	"evidence-guardian/internal/ocr"
	"evidence-guardian/internal/storage"
	"evidence-guardian/internal/trigger"
	"evidence-guardian/ui/server"
	"evidence-guardian/ui/tray"
	"github.com/kbinani/screenshot"
)

func main() {
	decryptCmd := flag.String("decrypt", "", "解密证据文件: -decrypt=口令 -dir=证据目录")
	decryptDir := flag.String("dir", "./evidence", "待解密的证据目录")
	ocrTest := flag.Bool("ocr-test", false, "测试OCR识别率: 截图并显示识别结果")
	ocrBench := flag.Bool("ocr-bench", false, "OCR CPU开销基准测试: 连续截图+OCR并统计CPU/耗时")
	flag.Parse()

	if *ocrBench {
		runOCRBenchmark()
		return
	}

	if *ocrTest {
		runOCRTest()
		return
	}

	if *decryptCmd != "" {
		runDecrypt(*decryptCmd, *decryptDir)
		return
	}

	exeDir := filepath.Dir(os.Args[0])
	cfgPath := filepath.Join(exeDir, "config.yaml")
	logFile, err := os.OpenFile(
		filepath.Join(exeDir, "evidence-guardian.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644,
	)
	if err == nil {
		log.SetOutput(io.MultiWriter(logFile, os.Stderr))
		defer logFile.Close()
	} else {
		log.SetOutput(os.Stderr)
	}

	log.Println("证据卫士 v0.1.0 — 劳动者权益保护取证系统")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// Decrypt passphrase if it was stored encrypted
	if cfg.Storage.Passphrase != "" {
		if dec, err := base64.StdEncoding.DecodeString(cfg.Storage.Passphrase); err == nil {
			if plain, err := crypto.Unprotect(dec); err == nil {
				cfg.Storage.Passphrase = string(plain)
			}
		}
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

func runDecrypt(passphrase, dir string) {
	fmt.Println("证据卫士 — 批量解密工具")
	fmt.Println(strings.Repeat("-", 40))

	// Detect encryption method
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

func runOCRTest() {
	fmt.Println("证据卫士 — OCR 识别率测试")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("正在截取全屏并运行 OCR...\n")

	engine := ocr.New()
	if !engine.IsReady() {
		fmt.Println("❌ OCR 引擎不可用")
		return
	}
	defer engine.Close()

	text, err := engine.RecognizeDesktop()
	if err != nil {
		fmt.Printf("❌ OCR 识别失败: %v\n", err)
		return
	}

	fmt.Println("📄 识别结果:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println(text)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("\n共识别 %d 个字\n", len([]rune(text)))
}

func runOCRBenchmark() {
	fmt.Println("证据卫士 — OCR CPU 开销基准测试")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()
	fmt.Println("测试场景: 全屏截图 → Tesseract OCR 识别")
	fmt.Println("模拟频率: 每 10 秒一次")
	fmt.Println()

	engine := ocr.New()
	if !engine.IsReady() {
		fmt.Println("❌ Tesseract 未安装，请先安装: https://github.com/UB-Mannheim/tesseract/wiki")
		fmt.Println("   安装时勾选中文简体 (chi_sim)")
		return
	}
	defer engine.Close()

	const rounds = 5
	var totalOCR time.Duration
	var totalScreenshot time.Duration
	var totalChars int

	for i := 1; i <= rounds; i++ {
		fmt.Printf("[%d/%d] 截图中...\n", i, rounds)

		t0 := time.Now()
		n := screenshot.NumActiveDisplays()
		if n == 0 { continue }
		bounds := screenshot.GetDisplayBounds(0)
		for di := 1; di < n; di++ {
			b := screenshot.GetDisplayBounds(di)
			bounds = bounds.Union(b)
		}
		img, err := screenshot.CaptureRect(bounds)
		t1 := time.Now()
		if err != nil {
			fmt.Printf("  截图失败: %v\n", err)
			continue
		}
		totalScreenshot += t1.Sub(t0)

		text, err := engine.Recognize(img)
		t2 := time.Now()
		if err != nil {
			fmt.Printf("  OCR失败: %v\n", err)
			continue
		}

		ocrDur := t2.Sub(t1)
		totalOCR += ocrDur
		chars := len([]rune(text))
		totalChars += chars

		fmt.Printf("  截图:%v  OCR:%v  识别%d字  CPU等效:%.1f%%\n",
			t1.Sub(t0).Round(time.Millisecond),
			ocrDur.Round(time.Millisecond),
			chars,
			float64(ocrDur.Milliseconds())/10000*100)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("📊 测试结果汇总")
	fmt.Println(strings.Repeat("-", 60))

	avgOCR := totalOCR / rounds
	avgSS := totalScreenshot / rounds

	fmt.Printf("平均截图耗时:     %v\n", avgSS.Round(time.Millisecond))
	fmt.Printf("平均OCR耗时:      %v\n", avgOCR.Round(time.Millisecond))
	fmt.Printf("平均识别字数:     %d 字\n", totalChars/rounds)
	fmt.Println()

	usage := float64(avgOCR.Milliseconds()) / 10000 * 100
	peakUsage := 0.0
	for i := 1; i <= rounds; i++ {
		_ = i
	}
	_ = peakUsage

	fmt.Println("🔍 10秒间隔CPU开销评估:")
	if usage < 5 {
		fmt.Printf("  ✅ CPU开销 %.1f%%，极低，10秒间隔完全可行\n", usage)
	} else if usage < 15 {
		fmt.Printf("  ⚠️ CPU开销 %.1f%%，中等，10秒间隔可行\n", usage)
	} else if usage < 30 {
		fmt.Printf("  ⚠️ CPU开销 %.1f%%，较高，建议间隔延长至30秒\n", usage)
	} else {
		fmt.Printf("  ❌ CPU开销 %.1f%%，过高，建议仅手动触发\n", usage)
	}
}


