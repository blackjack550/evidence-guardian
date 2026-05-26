package ocr

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kbinani/screenshot"
)

type Match struct {
	Text    string
	Keyword string
}

type Engine struct {
	ready    bool
	tesseractPath string
}

func New() *Engine {
	e := &Engine{}
	path := findTesseract()
	if path == "" {
		log.Println("Tesseract OCR 未安装")
		log.Println("下载: https://github.com/UB-Mannheim/tesseract/wiki")
		log.Println("安装时勾选中文简体语言包 (chi_sim)")
		return e
	}
	e.tesseractPath = path
	e.ready = testTesseract(path)
	if e.ready {
		log.Printf("OCR引擎已就绪: %s", path)
	}
	return e
}

func (e *Engine) IsReady() bool { return e.ready }

func findTesseract() string {
	paths := []string{
		"tesseract",
		"C:\\Program Files\\Tesseract-OCR\\tesseract.exe",
		"C:\\Program Files (x86)\\Tesseract-OCR\\tesseract.exe",
		fmt.Sprintf("%s\\Program Files\\Tesseract-OCR\\tesseract.exe", os.Getenv("SYSTEMDRIVE")),
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

func testTesseract(path string) bool {
	cmd := exec.Command(path, "--list-langs")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	hasChinese := strings.Contains(strings.ToLower(string(out)), "chi_sim")
	if !hasChinese {
		log.Println("Tesseract 缺少中文语言包 (chi_sim)")
	}
	return hasChinese
}

func (e *Engine) Recognize(img *image.RGBA) (string, error) {
	tmpDir := os.TempDir()
	imgPath := filepath.Join(tmpDir, "ev_ocr_input.png")
	outPath := filepath.Join(tmpDir, "ev_ocr_output")

	f, err := os.Create(imgPath)
	if err != nil {
		return "", err
	}
	png.Encode(f, img)
	f.Close()
	defer os.Remove(imgPath)
	defer os.Remove(outPath + ".txt")

	cmd := exec.Command(e.tesseractPath, imgPath, outPath,
		"-l", "chi_sim+eng", "--psm", "3")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Env = append(os.Environ(), "TESSDATA_PREFIX="+os.Getenv("TESSDATA_PREFIX"))
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Tesseract 识别失败: %w", err)
	}

	data, err := os.ReadFile(outPath + ".txt")
	if err != nil {
		return "", fmt.Errorf("读取OCR结果失败: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

func (e *Engine) RecognizeDesktop() (string, error) {
	if !e.ready {
		return "", fmt.Errorf("OCR未就绪")
	}
	img, err := captureDesktop()
	if err != nil {
		return "", err
	}
	return e.Recognize(img)
}

func (e *Engine) ScanDesktop(keywords []string) ([]Match, error) {
	if !e.ready {
		return nil, fmt.Errorf("OCR未就绪")
	}
	img, err := captureDesktop()
	if err != nil {
		return nil, err
	}
	text, err := e.Recognize(img)
	if err != nil {
		return nil, err
	}
	textLower := strings.ToLower(text)
	var matches []Match
	for _, kw := range keywords {
		if strings.Contains(textLower, strings.ToLower(kw)) {
			matches = append(matches, Match{Text: text, Keyword: kw})
		}
	}
	return matches, nil
}

func captureDesktop() (*image.RGBA, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return nil, fmt.Errorf("无显示器")
	}
	bounds := screenshot.GetDisplayBounds(0)
	for i := 1; i < n; i++ {
		b := screenshot.GetDisplayBounds(i)
		bounds = bounds.Union(b)
	}
	return screenshot.CaptureRect(bounds)
}

func (e *Engine) Close() {}
