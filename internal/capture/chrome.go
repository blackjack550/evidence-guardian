package capture

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type chromeDetector struct {
	name    string
	process string
}

func newChromeDetector(name, process string) *chromeDetector {
	return &chromeDetector{name: name, process: process}
}

func (c *chromeDetector) Name() string { return c.name }

func (c *chromeDetector) GetTabs(ctx context.Context) ([]TabInfo, error) {
	allocCtx, cancel := c.connect(ctx)
	if cancel != nil {
		defer cancel()
	} else {
		// Only try to connect, never auto-launch browser
		return nil, fmt.Errorf("%s 未开启调试端口", c.name)
	}

	tabsCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var infos []*target.Info
	if err := chromedp.Run(tabsCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		ts, err := chromedp.Targets(ctx)
		if err != nil {
			return err
		}
		infos = ts
		return nil
	})); err != nil {
		return nil, fmt.Errorf("获取标签页失败: %w", err)
	}

	var tabs []TabInfo
	for _, info := range infos {
		if info.Type != "page" {
			continue
		}
		tab := TabInfo{Title: info.Title, URL: info.URL}

		pageCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithTargetID(info.TargetID))
		func() {
			defer cancel()
			timeoutCtx, cancel := context.WithTimeout(pageCtx, 5*time.Second)
			defer cancel()
			var text string
			if err := chromedp.Run(timeoutCtx,
				chromedp.Text("body", &text, chromedp.ByQuery),
			); err == nil {
				tab.Text = strings.TrimSpace(text)
			}
		}()

		tabs = append(tabs, tab)
	}

	return tabs, nil
}

func (c *chromeDetector) findBrowserExe() string {
	for _, p := range c.findBrowserPaths() {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (c *chromeDetector) findBrowserPaths() []string {
	localAppData := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	userProfile := os.Getenv("USERPROFILE")

	switch c.process {
	case "chrome.exe":
		return []string{
			programFiles + `\Google\Chrome\Application\chrome.exe`,
			programFilesX86 + `\Google\Chrome\Application\chrome.exe`,
			localAppData + `\Google\Chrome\Application\chrome.exe`,
			userProfile + `\AppData\Local\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
	case "msedge.exe":
		return []string{
			programFilesX86 + `\Microsoft\Edge\Application\msedge.exe`,
			programFiles + `\Microsoft\Edge\Application\msedge.exe`,
			localAppData + `\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
			`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		}
	default:
		return []string{c.process}
	}
}

func (c *chromeDetector) connect(ctx context.Context) (context.Context, context.CancelFunc) {
	for _, port := range []int{9222, 9223, 9224, 9225} {
		if checkPort(port) {
			url := fmt.Sprintf("ws://127.0.0.1:%d", port)
			allocCtx, cancel := chromedp.NewRemoteAllocator(ctx, url)
			log.Printf("[%s] 连接到远端实例端口 %d", c.name, port)
			return allocCtx, cancel
		}
	}
	return nil, nil
}

func (c *chromeDetector) launch(ctx context.Context) (context.Context, context.CancelFunc) {
	port := findFreePort()
	if port == 0 {
		return nil, nil
	}

	exePath := c.findBrowserExe()
	if exePath == "" {
		return nil, nil
	}

	userDataDir := os.TempDir() + fmt.Sprintf("\\evidence_guardian_chrome_%d", port)
	cmd := exec.Command(exePath,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--no-first-run",
		"--new-window", "about:blank")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Start(); err != nil {
		os.RemoveAll(userDataDir)
		return nil, nil
	}

	for i := 0; i < 30; i++ {
		if checkPort(port) {
			url := fmt.Sprintf("ws://127.0.0.1:%d", port)
			allocCtx, cancel := chromedp.NewRemoteAllocator(ctx, url)
			log.Printf("[%s] 已启动进程，端口 %d", c.name, port)
			return allocCtx, cancel
		}
		time.Sleep(200 * time.Millisecond)
	}

	return nil, nil
}
