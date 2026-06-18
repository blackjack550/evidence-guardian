package icon

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
)

func Generate() []byte {
	w, h := 32, 32
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if inShield(x, y) {
				img.Set(x, y, color.RGBA{26, 115, 232, 255})
			}
		}
	}

	// White checkmark
	for _, p := range [][2]int{
		{10, 15}, {11, 15}, {12, 16}, {13, 16},
		{14, 17}, {15, 17}, {16, 16}, {17, 16},
		{18, 15}, {19, 15}, {20, 14}, {21, 14},
	} {
		img.Set(p[0], p[1], color.RGBA{255, 255, 255, 255})
	}

	return buildBMPICO(img, w, h)
}

func buildBMPICO(img *image.RGBA, w, h int) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint16(0))  // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1))  // type=ICO
	binary.Write(&buf, binary.LittleEndian, uint16(1))  // count

	// bmpData = BITMAPINFOHEADER(40) + BGRA pixels + AND mask
	bmpHeader := make([]byte, 40)
	binary.LittleEndian.PutUint32(bmpHeader[0:], 40)          // biSize
	binary.LittleEndian.PutUint32(bmpHeader[4:], uint32(w))   // biWidth
	binary.LittleEndian.PutUint32(bmpHeader[8:], uint32(h*2)) // biHeight (doubled for ICO)
	binary.LittleEndian.PutUint16(bmpHeader[12:], 1)          // biPlanes
	binary.LittleEndian.PutUint16(bmpHeader[14:], 32)         // biBitCount
	// biCompression=0, etc. are already zero

	// BGRA pixel data (bottom-up)
	pixels := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := img.At(x, h-1-y).RGBA()
			idx := (y*w + x) * 4
			pixels[idx+0] = byte(b >> 8)   // B
			pixels[idx+1] = byte(g >> 8)   // G
			pixels[idx+2] = byte(r >> 8)   // R
			pixels[idx+3] = byte(a >> 8)   // A
		}
	}
	// AND mask (all zeros for 32bpp)
	andMask := make([]byte, ((w+31)/32*4)*h)

	// Directory entry
	offset := 6 + 16 // header + entry
	bmpSize := len(bmpHeader) + len(pixels) + len(andMask)
	buf.WriteByte(byte(w))
	buf.WriteByte(byte(h))
	buf.WriteByte(0) // colors
	buf.WriteByte(0) // reserved
	binary.Write(&buf, binary.LittleEndian, uint16(1))  // planes=1 (BMP)
	binary.Write(&buf, binary.LittleEndian, uint16(32)) // bpp=32
	binary.Write(&buf, binary.LittleEndian, uint32(bmpSize))
	binary.Write(&buf, binary.LittleEndian, uint32(offset))

	// BMP data
	buf.Write(bmpHeader)
	buf.Write(pixels)
	buf.Write(andMask)
	return buf.Bytes()
}

func inShield(x, y int) bool {
	cx, cy := x-16, y-16
	if cy < -13 || cy > 13 {
		return false
	}
	w := 13
	if cy > 0 {
		w = 13 - cy/2
	}
	return cx >= -w && cx <= w && !(cx*cx+cy*cy < 6)
}
