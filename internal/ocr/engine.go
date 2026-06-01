package ocr

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

type Match struct {
	Text    string
	Keyword string
}

type Engine struct {
	ready         bool
	tesseractPath string
}

func New() *Engine {
	e := &Engine{}
	path := findTesseract()
	if path == "" {
		log.Println("Tesseract OCR 未安装")
		log.Println("下载: https://github.com/UB-Mannheim/tesseract/wiki")
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
func (e *Engine) Close()        {}

func getTessdataPrefix() string {
	if p := os.Getenv("TESSDATA_PREFIX"); p != "" {
		return p
	}
	// Check bundled tesseract/tessdata next to exe
	exeDir := filepath.Dir(os.Args[0])
	bundled := filepath.Join(exeDir, "tesseract", "tessdata")
	if _, err := os.Stat(filepath.Join(bundled, "chi_sim.traineddata")); err == nil {
		return bundled
	}
	// Check default user profile location
	userPrefix := os.Getenv("USERPROFILE") + "\\.tesseract\\tessdata"
	if _, err := os.Stat(userPrefix + "\\chi_sim.traineddata"); err == nil {
		return userPrefix
	}
	// Check install directory
	progPrefix := "C:\\Program Files\\Tesseract-OCR\\tessdata"
	if _, err := os.Stat(progPrefix + "\\chi_sim.traineddata"); err == nil {
		return progPrefix
	}
	return ""
}

func findTesseract() string {
	// First check app directory (bundled portable)
	exeDir := filepath.Dir(os.Args[0])
	bundled := filepath.Join(exeDir, "tesseract", "tesseract.exe")
	if _, err := os.Stat(bundled); err == nil {
		return bundled
	}
	// Then common locations
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

func findTessdata(exeDir string) string {
	tessDir := filepath.Join(exeDir, "tesseract", "tessdata")
	if _, err := os.Stat(filepath.Join(tessDir, "chi_sim.traineddata")); err == nil {
		return tessDir
	}
	return ""
}

func tessEnv() string {
	p := getTessdataPrefix()
	if p != "" {
		return "TESSDATA_PREFIX=" + p
	}
	return ""
}

func testTesseract(path string) bool {
	cmd := exec.Command(path, "--list-langs")
	if e := tessEnv(); e != "" {
		cmd.Env = append(os.Environ(), e)
	}
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

func (e *Engine) RecognizeBytes(imgData []byte) (string, error) {
	tmpDir := os.TempDir()
	imgPath := filepath.Join(tmpDir, "ev_ocr_input.png")
	outPath := filepath.Join(tmpDir, "ev_ocr_output")

	if err := os.WriteFile(imgPath, imgData, 0644); err != nil {
		return "", err
	}
	defer os.Remove(imgPath)
	defer os.Remove(outPath + ".txt")

	cmd := exec.Command(e.tesseractPath, imgPath, outPath,
		"-l", "chi_sim+eng", "--psm", "3")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if e := tessEnv(); e != "" {
		cmd.Env = append(os.Environ(), e)
	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Tesseract识别失败: %w", err)
	}

	data, err := os.ReadFile(outPath + ".txt")
	if err != nil {
		return "", fmt.Errorf("读取OCR结果失败: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}
