package icon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

func Generate() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	shieldColor := color.RGBA{R: 26, G: 115, B: 232, A: 255}
	bgColor := color.RGBA{0, 0, 0, 0}

	// Draw shield shape
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			if isInShield(x, y) {
				img.Set(x, y, shieldColor)
			} else {
				img.Set(x, y, bgColor)
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func isInShield(x, y int) bool {
	// Map to -32..32 coordinate space
	cx, cy := x-32, y-32
	if cy < -28 || cy > 28 {
		return false
	}

	halfWidth := 28
	if cy > 0 {
		// Bottom half: narrow trapezoid
		halfWidth = 28 - cy/2
	}

	if cx < -halfWidth || cx > halfWidth {
		return false
	}

	// Top curve
	if cy < -20 {
		dy := cy + 28
		limit := 28 - dy*dy/8
		if cx < -limit || cx > limit {
			return false
		}
	}

	// Checkmark inside shield
	if cy > -5 && cy < 15 && cx > -10 && cx < 20 {
		if cy > cx/2+5 && cy < cx/2+8 {
			return true
		}
	}

	return true
}
