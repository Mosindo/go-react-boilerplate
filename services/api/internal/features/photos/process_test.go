package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func makePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func TestProcessResizesAndReencodesAsJPEG(t *testing.T) {
	out, err := Process(makePNG(2000, 1000))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("output is not a JPEG: %v", err)
	}
	if cfg.Width != 1280 || cfg.Height != 640 {
		t.Fatalf("unexpected size %dx%d", cfg.Width, cfg.Height)
	}
}

func TestProcessKeepsSmallImages(t *testing.T) {
	out, err := Process(makePNG(100, 80))
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := jpeg.DecodeConfig(bytes.NewReader(out))
	if cfg.Width != 100 || cfg.Height != 80 {
		t.Fatalf("unexpected size %dx%d", cfg.Width, cfg.Height)
	}
}

func TestProcessRejectsNonImages(t *testing.T) {
	cases := map[string][]byte{
		"text":   []byte("hello world, not an image"),
		"html":   []byte("<html><script>alert(1)</script></html>"),
		"empty":  {},
		"gif":    []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"badpng": append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 32)...),
	}
	for name, data := range cases {
		if _, err := Process(data); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestProcessRejectsHugeDimensions(t *testing.T) {
	// A tiny PNG header declaring 20000x20000 pixels must be refused before decoding.
	raw := makePNG(10, 10)
	huge := append([]byte{}, raw...)
	// IHDR width/height live at bytes 16..24; CRC will mismatch but DecodeConfig does not verify it.
	copy(huge[16:24], []byte{0, 0, 0x4e, 0x20, 0, 0, 0x4e, 0x20})
	if _, err := Process(huge); err == nil {
		t.Fatal("expected oversize rejection")
	}
}

func TestReadLimited(t *testing.T) {
	if _, err := ReadLimited(bytes.NewReader(make([]byte, 11)), 10); err == nil {
		t.Fatal("expected size error")
	}
	if _, err := ReadLimited(bytes.NewReader(make([]byte, 10)), 10); err != nil {
		t.Fatal(err)
	}
}
