package capture

import (
	"fmt"
	"image"
	"os/exec"
)

type VideoConfig struct {
	DurationSec int
	FPS         int
	Width       int
	Height      int
	OutputPath  string
}

type VideoRecorder interface {
	Start(cfg VideoConfig) error
	WriteFrame(img *image.RGBA) error
	Stop() error
}

type FFmpegRecorder struct {
	cmd    *exec.Cmd
	stdin  pipeWriter
	frameCh chan *image.RGBA
	done   chan error
}

type pipeWriter interface {
	Write([]byte) (int, error)
	Close() error
}

func NewFFmpegRecorder(ffmpegPath string) (*FFmpegRecorder, error) {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &FFmpegRecorder{
		frameCh: make(chan *image.RGBA, 60),
		done:    make(chan error, 1),
	}, nil
}

func (r *FFmpegRecorder) Start(cfg VideoConfig) error {
	r.cmd = exec.Command("ffmpeg",
		"-f", "rawvideo",
		"-pix_fmt", "rgba",
		"-s", fmt.Sprintf("%dx%d", cfg.Width, cfg.Height),
		"-r", fmt.Sprintf("%d", cfg.FPS),
		"-i", "-",
		"-c:v", "libx264",
		"-preset", "fast",
		"-pix_fmt", "yuv420p",
		"-y",
		cfg.OutputPath,
	)

	stdin, err := r.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建FFmpeg管道失败: %w", err)
	}
	r.stdin = stdin

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("启动FFmpeg失败: %w", err)
	}

	go r.writeLoop()
	return nil
}

func (r *FFmpegRecorder) WriteFrame(img *image.RGBA) error {
	r.frameCh <- img
	return nil
}

func (r *FFmpegRecorder) writeLoop() {
	for img := range r.frameCh {
		r.stdin.Write(img.Pix)
	}
	r.stdin.Close()
}

func (r *FFmpegRecorder) Stop() error {
	close(r.frameCh)
	return r.cmd.Wait()
}
