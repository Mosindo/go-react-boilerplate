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

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestProcessImageResizesAndReencodes(t *testing.T) {
	out, err := ProcessImage(makePNG(t, 2000, 1000))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1440 || out.Height != 720 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Full)); err != nil {
		t.Fatalf("full variant is not a jpeg: %v", err)
	}
	thumb, err := jpeg.DecodeConfig(bytes.NewReader(out.Thumb))
	if err != nil || thumb.Width != 480 {
		t.Fatalf("unexpected thumb %+v err=%v", thumb, err)
	}
}

func TestProcessImageRejectsDangerousInput(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want error
	}{
		"html disguised": {[]byte("<html><script>alert(1)</script></html>"), ErrUnsupportedType},
		"svg":            {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), ErrUnsupportedType},
		"empty":          {nil, ErrImageCorrupt},
		"too small":      {makePNG(t, 50, 50), ErrImageTooSmall},
		"truncated png":  {makePNG(t, 300, 300)[:100], ErrImageCorrupt},
		"too large":      {make([]byte, MaxUploadBytes+1), ErrImageTooLarge},
	}
	for name, tc := range cases {
		if _, err := ProcessImage(tc.data); !errors.Is(err, tc.want) {
			t.Fatalf("%s: expected %v, got %v", name, tc.want, err)
		}
	}
}

func TestApplyOrientationRotates(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	rotated := applyOrientation(img, 6) // 90° clockwise
	if rotated.Bounds().Dx() != 2 || rotated.Bounds().Dy() != 4 {
		t.Fatalf("unexpected bounds %v", rotated.Bounds())
	}
	r, _, _, _ := rotated.At(1, 0).RGBA()
	if r == 0 {
		t.Fatal("top-left pixel should move to top-right after 90° rotation")
	}
}

func TestJPEGOrientationParsesExif(t *testing.T) {
	// Minimal JPEG: SOI + APP1(Exif, little endian, 1 IFD entry orientation=6) + EOI.
	exif := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 0x2A, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	segLen := len(exif) + 2
	data := []byte{0xFF, 0xD8, 0xFF, 0xE1, byte(segLen >> 8), byte(segLen)}
	data = append(data, exif...)
	data = append(data, 0xFF, 0xD9)
	if got := jpegOrientation(data); got != 6 {
		t.Fatalf("expected orientation 6, got %d", got)
	}
	if got := jpegOrientation([]byte{0xFF, 0xD8, 0xFF, 0xD9}); got != 1 {
		t.Fatalf("expected default orientation, got %d", got)
	}
}
