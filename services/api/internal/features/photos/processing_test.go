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
		img.Set(x, 0, color.RGBA{R: 255, A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessImageReencodesAndResizes(t *testing.T) {
	out, err := ProcessImage(encodePNG(t, 3200, 1600))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1600 || out.Height != 800 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, format, err := image.DecodeConfig(bytes.NewReader(out.Data)); err != nil || format != "jpeg" {
		t.Fatalf("expected jpeg output, got %q (%v)", format, err)
	}
}

func TestProcessImageRejectsDangerousInput(t *testing.T) {
	cases := map[string][]byte{
		"empty":     {},
		"html":      []byte("<html><script>alert(1)</script></html>"),
		"tiny":      encodePNG(t, 50, 50),
		"gif":       []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"truncated": encodePNG(t, 400, 400)[:60],
	}
	for name, data := range cases {
		if _, err := ProcessImage(data); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	if _, err := ProcessImage(make([]byte, MaxUploadBytes+1)); !errors.Is(err, ErrImageTooLarge) {
		t.Errorf("expected size rejection, got %v", err)
	}
}

func TestApplyOrientationRotates(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	src.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	rotated := applyOrientation(src, 6)
	if rotated.Bounds().Dx() != 2 || rotated.Bounds().Dy() != 4 {
		t.Fatalf("unexpected bounds %v", rotated.Bounds())
	}
	if rotated.RGBAAt(1, 0).R != 255 {
		t.Fatal("top-left pixel must move to top-right after a 90° clockwise rotation")
	}
}

func TestJPEGOrientationParsing(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatal(err)
	}
	plain := buf.Bytes()
	if got := jpegOrientation(plain); got != 1 {
		t.Fatalf("expected default orientation, got %d", got)
	}
	// Build an APP1 EXIF segment (big endian) with orientation = 6.
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 6, 0, 0, 0, 0, 0, 0}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	size := len(payload) + 2
	segment := append([]byte{0xFF, 0xE1, byte(size >> 8), byte(size)}, payload...)
	withExif := append(append([]byte{0xFF, 0xD8}, segment...), plain[2:]...)
	if got := jpegOrientation(withExif); got != 6 {
		t.Fatalf("expected orientation 6, got %d", got)
	}
}
