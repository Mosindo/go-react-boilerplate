package photos

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func encodeJPEGBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withOrientation injects an EXIF APP1 segment carrying the orientation tag.
func withOrientation(jpegData []byte, orientation uint16) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1}
	entry := make([]byte, 12)
	binary.BigEndian.PutUint16(entry[0:], 0x0112)
	binary.BigEndian.PutUint16(entry[2:], 3)
	binary.BigEndian.PutUint32(entry[4:], 1)
	binary.BigEndian.PutUint16(entry[8:], orientation)
	tiff = append(tiff, entry...)
	tiff = append(tiff, 0, 0, 0, 0)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}
	seg = append(seg, payload...)
	out := append([]byte{}, jpegData[:2]...)
	out = append(out, seg...)
	return append(out, jpegData[2:]...)
}

func TestProcessResizesAndReencodes(t *testing.T) {
	data := encodeJPEGBytes(t, solid(3200, 2000, color.RGBA{200, 10, 10, 255}))
	p, err := process(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.width != 1600 || p.height != 1000 {
		t.Fatalf("expected 1600x1000, got %dx%d", p.width, p.height)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(p.thumb))
	if err != nil || cfg.Width != 400 {
		t.Fatalf("thumb: %v %v", cfg, err)
	}
}

func TestProcessAcceptsPNGAndNeverUpscales(t *testing.T) {
	var buf bytes.Buffer
	_ = png.Encode(&buf, solid(120, 80, color.RGBA{0, 0, 255, 128}))
	p, err := process(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if p.width != 120 || p.height != 80 {
		t.Fatalf("small images must not be upscaled, got %dx%d", p.width, p.height)
	}
}

func TestProcessRejectsNonImages(t *testing.T) {
	for name, data := range map[string][]byte{
		"text":    []byte("hello world"),
		"php":     []byte("<?php system($_GET['c']); ?>"),
		"svg":     []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"empty":   {},
		"gif":     append([]byte("GIF89a"), make([]byte, 40)...),
		"trunc":   {0xFF, 0xD8, 0xFF, 0xE0, 0, 16},
		"pe":      append([]byte("MZ"), make([]byte, 100)...),
		"jpegbad": append([]byte{0xFF, 0xD8, 0xFF}, bytes.Repeat([]byte{1}, 200)...),
	} {
		if _, err := process(data); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}

func TestProcessStripsExifPayload(t *testing.T) {
	data := withOrientation(encodeJPEGBytes(t, solid(64, 32, color.White)), 1)
	if !bytes.Contains(data, []byte("Exif")) {
		t.Fatal("test fixture should contain EXIF")
	}
	p, err := process(data)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(p.full, []byte("Exif")) {
		t.Fatal("metadata must be stripped from the stored image")
	}
}

func TestOrientationIsApplied(t *testing.T) {
	// 64x32 landscape with orientation 6 (rotate 90° CW) must become 32x64 portrait.
	data := withOrientation(encodeJPEGBytes(t, solid(64, 32, color.RGBA{10, 200, 10, 255})), 6)
	if got := jpegOrientation(data); got != 6 {
		t.Fatalf("orientation parsed as %d", got)
	}
	p, err := process(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.width != 32 || p.height != 64 {
		t.Fatalf("expected 32x64 after rotation, got %dx%d", p.width, p.height)
	}
	if got := jpegOrientation(encodeJPEGBytes(t, solid(8, 8, color.White))); got != 1 {
		t.Fatalf("missing EXIF must default to 1, got %d", got)
	}
}

func TestLocalStorageRejectsTraversal(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../etc/passwd", "a/b.jpg", "", "x.jpg", "00000000-0000-4000-8000-000000000000.png"} {
		if err := s.Put(key, []byte("x")); err == nil {
			t.Errorf("key %q must be rejected", key)
		}
	}
	key := "00000000-0000-4000-8000-000000000000.jpg"
	if err := s.Put(key, []byte("data")); err != nil {
		t.Fatal(err)
	}
	if b, err := s.Get(key); err != nil || string(b) != "data" {
		t.Fatalf("get: %q %v", b, err)
	}
	if err := s.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(key); err != ErrStorageNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}
