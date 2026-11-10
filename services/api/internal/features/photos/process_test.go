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

func TestProcessReencodesAndResizes(t *testing.T) {
	out, err := Process(encodePNG(t, 2000, 1000))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != MaxOutputSide || out.Height != MaxOutputSide/2 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be a JPEG: %v", err)
	}
}

func TestProcessRejectsDangerousInput(t *testing.T) {
	cases := map[string][]byte{
		"html":      []byte("<html><script>alert(1)</script></html>"),
		"php":       []byte("<?php system($_GET['c']); ?>"),
		"truncated": encodePNG(t, 300, 300)[:100],
		"svg":       []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>`),
	}
	for name, data := range cases {
		if _, err := Process(data); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	if _, err := Process(encodePNG(t, 100, 100)); !errors.Is(err, ErrImageTooSmall) {
		t.Errorf("expected too small, got %v", err)
	}
}

func TestProcessStripsTrailingPayload(t *testing.T) {
	polyglot := append(encodePNG(t, 300, 300), []byte("<?php echo 'pwned'; ?>")...)
	out, err := Process(polyglot)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if bytes.Contains(out.Data, []byte("pwned")) {
		t.Fatal("trailing payload must not survive re-encoding")
	}
}

func TestFit(t *testing.T) {
	if w, h := fit(800, 600, 1440); w != 800 || h != 600 {
		t.Fatal("small images must not be upscaled")
	}
	if w, h := fit(1000, 3000, 1440); w != 480 || h != 1440 {
		t.Fatalf("unexpected portrait fit %dx%d", w, h)
	}
}
