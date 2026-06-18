package capture

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/kbinani/screenshot"
)

type Recorder struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	stdin    *os.File
	width    int
	height   int
	fps      int
	duration time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func NewRecorder(width, height, fps int, durationSec int) *Recorder {
	return &Recorder{
		width:    width,
		height:   height,
		fps:      fps,
		duration: time.Duration(durationSec) * time.Second,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (r *Recorder) Start(outputPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	os.MkdirAll(filepath.Dir(outputPath), 0755)

	// Find ffmpeg
	ffmpegPath := findFFmpeg()
	if ffmpegPath == "" {
		return fmt.Errorf("未找到FFmpeg，请安装后再试")
	}

	r.cmd = exec.Command(ffmpegPath,
		"-y",
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%dx%d", r.width, r.height),
		"-r", fmt.Sprintf("%d", r.fps),
		"-i", "-",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-crf", "28",
		"-pix_fmt", "yuv420p",
		outputPath,
	)

	stdin, err := r.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建管道失败: %w", err)
	}

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("启动FFmpeg失败: %w", err)
	}

	r.stdin = stdin.(*os.File)

	go r.record()
	return nil
}

func (r *Recorder) record() {
	defer func() {
		r.stdin.Close()
		r.cmd.Wait()
		close(r.doneCh)
	}()

	frameInterval := time.Second / time.Duration(r.fps)
	totalFrames := int(r.duration / frameInterval)
	bounds := screenshot.GetDisplayBounds(0)
	n := screenshot.NumActiveDisplays()
	for i := 1; i < n; i++ {
		b := screenshot.GetDisplayBounds(i)
		bounds = bounds.Union(b)
	}

	for i := 0; i < totalFrames; i++ {
		select {
		case <-r.stopCh:
			return
		default:
		}

		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			log.Printf("录帧失败: %v", err)
			continue
		}

		r.stdin.Write(img.Pix)

		time.Sleep(frameInterval)
	}
}

func (r *Recorder) Stop() {
	close(r.stopCh)
	<-r.doneCh
}

func findFFmpeg() string {
	// First check app directory (bundled)
	exeDir := filepath.Dir(os.Args[0])
	bundled := filepath.Join(exeDir, "ffmpeg.exe")
	if _, err := os.Stat(bundled); err == nil {
		return bundled
	}
	// Then common locations
	paths := []string{
		"ffmpeg",
		"C:\\ffmpeg\\bin\\ffmpeg.exe",
		"C:\\Program Files\\ffmpeg\\bin\\ffmpeg.exe",
		"C:\\Program Files (x86)\\ffmpeg\\bin\\ffmpeg.exe",
	}
	for _, p := range paths {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func RecordDesktop(dir, name string, durationSec, fps int) (string, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return "", fmt.Errorf("无显示器")
	}

	bounds := screenshot.GetDisplayBounds(0)
	for i := 1; i < n; i++ {
		b := screenshot.GetDisplayBounds(i)
		bounds = bounds.Union(b)
	}

	w, h := bounds.Dx(), bounds.Dy()
	rec := NewRecorder(w, h, fps, durationSec)

	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, fmt.Sprintf("%s_%d.mp4", name, time.Now().UnixMilli()))

	if err := rec.Start(path); err != nil {
		return "", err
	}

	rec.Stop()
	return path, nil
}

func RecordWindow(rect RECT, dir, name string, durationSec, fps int) (string, error) {
	w := int(rect.Right - rect.Left)
	h := int(rect.Bottom - rect.Top)
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("无效窗口尺寸")
	}

	rec := NewRecorder(w, h, fps, durationSec)

	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, fmt.Sprintf("%s_%d.mp4", name, time.Now().UnixMilli()))

	if err := rec.Start(path); err != nil {
		return "", err
	}
	rec.Stop()
	return path, nil
}


