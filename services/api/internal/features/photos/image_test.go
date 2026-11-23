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
		img.Set(x, x%h, color.RGBA{200, 80, 60, 255})
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
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1600 || out.Height != 800 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be JPEG: %v", err)
	}
}

func TestProcessImageStripsEXIF(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	raw := buf.Bytes()
	// Inject a fake APP1/EXIF segment right after SOI.
	exif := append([]byte{0xFF, 0xE1, 0x00, 0x10}, []byte("Exif\x00\x00GPSDATA!")...)
	withExif := append(append([]byte{}, raw[:2]...), append(exif, raw[2:]...)...)

	out, err := ProcessImage(withExif)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if bytes.Contains(out.Data, []byte("Exif")) || bytes.Contains(out.Data, []byte("GPSDATA")) {
		t.Fatal("EXIF metadata must be stripped")
	}
}

func TestProcessImageRejectsDangerousInput(t *testing.T) {
	cases := map[string][]byte{
		"empty":     {},
		"html":      []byte("<html><script>alert(1)</script></html>"),
		"svg":       []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"gif":       []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"truncated": encodePNG(t, 400, 400)[:100],
	}
	for name, data := range cases {
		if _, err := ProcessImage(data); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	if _, err := ProcessImage(encodePNG(t, 100, 100)); !errors.Is(err, ErrImageTooSmall) {
		t.Errorf("expected too small, got %v", err)
	}
	if _, err := ProcessImage(make([]byte, MaxUploadBytes+1)); !errors.Is(err, ErrImageTooLarge) {
		t.Errorf("expected too large, got %v", err)
	}
}
