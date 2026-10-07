package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	return img
}

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, solid(w, h)); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestProcessImageDownscalesAndReencodes(t *testing.T) {
	out, err := processImage(encodePNG(t, 3000, 2000))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1440 || out.Height != 960 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output is not a valid jpeg: %v", err)
	}
}

func TestProcessImageRejectsBadInput(t *testing.T) {
	cases := map[string][]byte{
		"empty":       nil,
		"text":        []byte("<html><script>alert(1)</script></html>"),
		"svg":         []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`),
		"gif header":  []byte("GIF89a\x01\x00\x01\x00"),
		"truncated":   encodePNG(t, 400, 400)[:60],
		"too small":   encodePNG(t, 50, 50),
		"php in file": append([]byte("<?php system($_GET['c']); ?>"), make([]byte, 100)...),
	}
	for name, raw := range cases {
		if _, err := processImage(raw); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestProcessImageRejectsOversize(t *testing.T) {
	raw := make([]byte, maxUploadBytes+1)
	copy(raw, []byte("\x89PNG\r\n\x1a\n"))
	if _, err := processImage(raw); err == nil {
		t.Fatal("expected size error")
	}
}
