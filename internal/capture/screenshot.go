package capture

import (
	"fmt"
	"image"
	"image/png"
	"os"
)

type ScreenshotResult struct {
	Path      string
	Width     int
	Height    int
	WindowInfo
}

type Screenshotter interface {
	Capture(hwnd uintptr, rect RECT) (*image.RGBA, error)
	CaptureFullScreen() (*image.RGBA, error)
}

func SaveScreenshot(img *image.RGBA, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建截图文件失败: %w", err)
	}
	defer f.Close()

	return png.Encode(f, img)
}
