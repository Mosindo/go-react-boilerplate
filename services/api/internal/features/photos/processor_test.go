package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	return img
}

func TestProcessJPEGResizesAndStripsMetadata(t *testing.T) {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, solid(2400, 1600), nil)
	// Append a fake trailing payload: it must not survive re-encoding.
	raw := append(buf.Bytes(), []byte("<?php evil ?>")...)
	out, err := Process(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1080 || out.Height != 720 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if bytes.Contains(out.Full, []byte("evil")) {
		t.Fatal("payload survived")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out.Thumb))
	if err != nil || cfg.Width != 360 {
		t.Fatalf("thumb: %v %+v", err, cfg)
	}
}

func TestProcessPNG(t *testing.T) {
	var buf bytes.Buffer
	_ = png.Encode(&buf, solid(500, 700))
	out, err := Process(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 500 || out.Height != 700 {
		t.Fatalf("small images must not be upscaled: %dx%d", out.Width, out.Height)
	}
}

func TestProcessRejects(t *testing.T) {
	var g bytes.Buffer
	_ = gif.Encode(&g, solid(300, 300), nil)
	var small bytes.Buffer
	_ = jpeg.Encode(&small, solid(50, 50), nil)
	cases := map[string][]byte{
		"text":   []byte("hello world, definitely not an image"),
		"gif":    g.Bytes(),
		"html":   []byte("<html><script>alert(1)</script></html>"),
		"small":  small.Bytes(),
		"empty":  {},
		"badjpg": append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 64)...),
	}
	for name, raw := range cases {
		if _, err := Process(raw); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}
