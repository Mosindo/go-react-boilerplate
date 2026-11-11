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

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		img.Set(x, x%h, color.RGBA{200, 80, 60, 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessResizesAndReencodesAsJPEG(t *testing.T) {
	out, err := Process(pngBytes(t, 3200, 2400))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1600 || out.Height != 1200 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output is not a jpeg: %v", err)
	}
}

func TestProcessRejectsDangerousInputs(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want error
	}{
		"html":       {[]byte("<html><script>alert(1)</script></html>"), ErrUnsupportedType},
		"svg":        {[]byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`), ErrUnsupportedType},
		"tiny":       {pngBytes(t, 50, 50), ErrImageTooSmall},
		"truncated":  {pngBytes(t, 400, 400)[:120], ErrCorruptImage},
		"executable": {append([]byte("MZ\x90\x00"), make([]byte, 200)...), ErrUnsupportedType},
	}
	for name, c := range cases {
		if _, err := Process(c.data); !errors.Is(err, c.want) {
			t.Errorf("%s: expected %v, got %v", name, c.want, err)
		}
	}
}

func TestFit(t *testing.T) {
	if w, h := fit(800, 600, 1600); w != 800 || h != 600 {
		t.Fatal("small image must not be upscaled")
	}
	if w, h := fit(1000, 4000, 1600); w != 400 || h != 1600 {
		t.Fatalf("portrait fit wrong: %dx%d", w, h)
	}
}
