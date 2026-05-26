package ocr

import (
	"log"
)

type Match struct {
	Text    string
	Keyword string
}

type Engine struct {
	ready bool
}

func New() *Engine {
	log.Println("OCR引擎需要Tesseract支持，当前不可用")
	return &Engine{ready: false}
}

func (e *Engine) IsReady() bool { return e.ready }
func (e *Engine) Close()        {}
