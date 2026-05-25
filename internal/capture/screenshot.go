package capture

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/kbinani/screenshot"
)

type ScreenshotResult struct {
	Path   string
	Width  int
	Height int
	WindowInfo
}

func CaptureDesktop() (*image.RGBA, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return nil, fmt.Errorf("没有检测到显示器")
	}

	bounds := screenshot.GetDisplayBounds(0)
	for i := 1; i < n; i++ {
		b := screenshot.GetDisplayBounds(i)
		bounds = bounds.Union(b)
	}

	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %w", err)
	}
	return img, nil
}

func CaptureWindow(hwnd uintptr, rect RECT) (*image.RGBA, error) {
	img, err := screenshot.CaptureRect(image.Rect(
		int(rect.Left), int(rect.Top),
		int(rect.Right), int(rect.Bottom),
	))
	if err != nil {
		return nil, fmt.Errorf("窗口截图失败: %w", err)
	}
	return img, nil
}

func SavePNG(img *image.RGBA, dir, name string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("%s_%d.png", name, time.Now().UnixMilli()))
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return path, png.Encode(f, img)
}
