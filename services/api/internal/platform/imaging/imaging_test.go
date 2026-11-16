package imaging

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
		img.Set(x, x%h, color.RGBA{R: 200, A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessNormalizesToJPEG(t *testing.T) {
	res, err := Process(encodePNG(t, 2400, 1200))
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 1600 || res.Height != 800 {
		t.Fatalf("expected 1600x800, got %dx%d", res.Width, res.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(res.Data)); err != nil {
		t.Fatalf("output must be JPEG: %v", err)
	}
}

func TestProcessRejectsInvalidInput(t *testing.T) {
	if _, err := Process([]byte("<?php system($_GET['x']); ?>")); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected unsupported type, got %v", err)
	}
	if _, err := Process(encodePNG(t, 100, 100)); !errors.Is(err, ErrTooSmall) {
		t.Fatalf("expected too small, got %v", err)
	}
	// Valid PNG signature but truncated body.
	if _, err := Process(encodePNG(t, 300, 300)[:40]); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("expected corrupt, got %v", err)
	}
	if _, err := Process(nil); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected empty upload to be rejected, got %v", err)
	}
}

func TestFit(t *testing.T) {
	if w, h := fit(800, 600, 1600); w != 800 || h != 600 {
		t.Fatal("small images keep their size")
	}
	if w, h := fit(1000, 4000, 1600); w != 400 || h != 1600 {
		t.Fatalf("portrait fit wrong: %dx%d", w, h)
	}
}
