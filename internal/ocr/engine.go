package ocr

import "image"

type Result struct {
	Text      string
	Confidence float64
	Keywords  []string
}

type Engine interface {
	Recognize(img *image.RGBA) (*Result, error)
}

type Config struct {
	Enabled     bool
	IntervalSec int
	Keywords    []string
}
