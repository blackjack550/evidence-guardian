package capture

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
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
		allocCtx, cancel = c.launch(ctx)
		if cancel == nil {
			return nil, fmt.Errorf("无法连接或启动%s", c.name)
		}
		defer cancel()
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

	paths := []string{
		fmt.Sprintf(`C:\Program Files\%s\Application\%s`, c.name, c.process),
		fmt.Sprintf(`C:\Program Files (x86)\%s\Application\%s`, c.name, c.process),
	}

	var cmd *exec.Cmd
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			cmd = exec.Command(path,
				fmt.Sprintf("--remote-debugging-port=%d", port),
				"--no-first-run", "--new-window", "about:blank")
			break
		}
	}
	if cmd == nil {
		cmd = exec.Command(c.process,
			fmt.Sprintf("--remote-debugging-port=%d", port),
			"--no-first-run", "--new-window", "about:blank")
	}

	if err := cmd.Start(); err != nil {
		log.Printf("[%s] 启动失败: %v", c.name, err)
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
