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
			img.Set(x, y, color.RGBA{200, uint8(x % 255), uint8(y % 255), 255})
		}
	}
	return img
}

func encodePNG(img image.Image) []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func TestProcessImageResizesAndReencodes(t *testing.T) {
	out, err := processImage(encodePNG(solid(2400, 1600)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1080 || out.Height != 720 {
		t.Fatalf("expected 1080x720, got %dx%d", out.Width, out.Height)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(out.Thumb))
	if err != nil || format != "jpeg" || cfg.Width != 360 {
		t.Fatalf("thumb must be a 360px jpeg: %v %s %d", err, format, cfg.Width)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Full)); err != nil {
		t.Fatalf("full must be a valid jpeg: %v", err)
	}
}

func TestProcessImageNeverUpscales(t *testing.T) {
	out, err := processImage(encodePNG(solid(400, 500)))
	if err != nil || out.Width != 400 || out.Height != 500 {
		t.Fatalf("small images keep their size: %v %dx%d", err, out.Width, out.Height)
	}
}

func TestProcessImageRejectsBadInput(t *testing.T) {
	var jpg bytes.Buffer
	_ = jpeg.Encode(&jpg, solid(300, 300), nil)
	truncated := jpg.Bytes()[:200]

	cases := map[string][]byte{
		"empty":        {},
		"text":         []byte("hello world, definitely not an image"),
		"php":          []byte("<?php system($_GET['c']); ?>"),
		"gif":          []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"truncated":    truncated,
		"too small":    encodePNG(solid(100, 100)),
		"svg":          []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"jpeg+payload": append([]byte("<?php "), jpg.Bytes()...),
	}
	for name, data := range cases {
		if _, err := processImage(data); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if _, err := processImage(make([]byte, maxUploadBytes+1)); err == nil {
		t.Error("oversized input must be rejected")
	}
}

func TestProcessImageStripsPayloadAfterImage(t *testing.T) {
	var jpg bytes.Buffer
	_ = jpeg.Encode(&jpg, solid(300, 300), nil)
	polyglot := append(jpg.Bytes(), []byte("<?php evil(); ?>")...)
	out, err := processImage(polyglot)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Full, []byte("<?php")) {
		t.Fatal("trailing payload must not survive re-encoding")
	}
}
