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
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
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
	out, err := Process(encodePNG(t, 2400, 1200))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1600 || out.Height != 800 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output is not a JPEG: %v", err)
	}
}

func TestProcessRejectsDangerousInput(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want error
	}{
		"empty":       {nil, ErrInvalidImage},
		"html":        {[]byte("<html><script>alert(1)</script></html>"), ErrUnsupportedType},
		"svg":         {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), ErrUnsupportedType},
		"tiny":        {encodePNG(t, 50, 50), ErrImageTooSmall},
		"truncated":   {encodePNG(t, 300, 300)[:100], ErrInvalidImage},
		"exe as .jpg": {append([]byte("MZ"), make([]byte, 600)...), ErrUnsupportedType},
	}
	for name, tc := range cases {
		if _, err := Process(tc.data); !errors.Is(err, tc.want) {
			t.Errorf("%s: expected %v, got %v", name, tc.want, err)
		}
	}
}

func TestApplyOrientationSwapsDimensions(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	rotated := applyOrientation(src, 6)
	if rotated.Bounds().Dx() != 2 || rotated.Bounds().Dy() != 4 {
		t.Fatalf("expected 2x4, got %v", rotated.Bounds())
	}
	if r, _, _, _ := rotated.At(1, 0).RGBA(); r == 0 {
		t.Fatal("expected top-left pixel to move to top-right after 90° rotation")
	}
}

func TestJPEGOrientationParsing(t *testing.T) {
	// Minimal JPEG header with an APP1 EXIF segment holding orientation=6 (big endian).
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 6, 0, 0, 0, 0, 0, 0}
	seg := append([]byte("Exif\x00\x00"), tiff...)
	size := len(seg) + 2
	data := []byte{0xFF, 0xD8, 0xFF, 0xE1, byte(size >> 8), byte(size)}
	data = append(data, seg...)
	data = append(data, 0xFF, 0xD9)
	if got := jpegOrientation(data); got != 6 {
		t.Fatalf("expected orientation 6, got %d", got)
	}
	if got := jpegOrientation([]byte{0x00, 0x01}); got != 1 {
		t.Fatalf("expected default orientation, got %d", got)
	}
}
