package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessImageResizesAndReencodes(t *testing.T) {
	out, err := processImage(encodePNG(t, 2000, 1000))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.width != 1440 || out.height != 720 {
		t.Fatalf("unexpected size %dx%d", out.width, out.height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.data)); err != nil {
		t.Fatalf("output must be a JPEG: %v", err)
	}
}

func TestProcessImageRejectsBadInput(t *testing.T) {
	if _, err := processImage([]byte("<?php system($_GET['x']); ?>")); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
	if _, err := processImage(encodePNG(t, 100, 100)); !errors.Is(err, ErrImageTooSmall) {
		t.Fatalf("expected ErrImageTooSmall, got %v", err)
	}
	// Valid PNG signature but truncated body.
	truncated := encodePNG(t, 300, 300)[:64]
	if _, err := processImage(truncated); err == nil {
		t.Fatal("expected truncated image to be rejected")
	}
}

func TestFitWithin(t *testing.T) {
	if w, h := fitWithin(800, 600, 1440); w != 800 || h != 600 {
		t.Fatal("small image must not be upscaled")
	}
	if w, h := fitWithin(1000, 3000, 1440); w != 480 || h != 1440 {
		t.Fatalf("unexpected portrait fit %dx%d", w, h)
	}
}
