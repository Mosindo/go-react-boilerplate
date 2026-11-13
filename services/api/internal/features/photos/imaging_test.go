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
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessUploadReencodesAndResizes(t *testing.T) {
	out, err := ProcessUpload(bytes.NewReader(encodePNG(t, 2400, 1200)))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1600 || out.Height != 800 {
		t.Fatalf("expected 1600x800, got %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be a JPEG: %v", err)
	}
}

func TestProcessUploadRejectsUnsafeInput(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want error
	}{
		{"empty", nil, ErrCorruptImage},
		{"html disguised", []byte("<html><script>alert(1)</script></html>"), ErrUnsupportedType},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script/></svg>`), ErrUnsupportedType},
		{"too small", encodePNG(t, 100, 100), ErrImageTooSmall},
		{"truncated png", encodePNG(t, 400, 400)[:200], ErrCorruptImage},
	}
	for _, tc := range cases {
		_, err := ProcessUpload(bytes.NewReader(tc.data))
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: expected %v, got %v", tc.name, tc.want, err)
		}
	}

	big := make([]byte, MaxUploadBytes+10)
	copy(big, encodePNG(t, 400, 400))
	if _, err := ProcessUpload(bytes.NewReader(big)); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestFitWithin(t *testing.T) {
	if w, h := fitWithin(800, 600, 1600); w != 800 || h != 600 {
		t.Fatalf("small images are untouched, got %dx%d", w, h)
	}
	if w, h := fitWithin(1000, 4000, 1600); w != 400 || h != 1600 {
		t.Fatalf("portrait fit, got %dx%d", w, h)
	}
}
