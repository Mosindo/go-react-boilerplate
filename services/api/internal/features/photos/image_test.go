package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func TestProcessImageRejectsNonImages(t *testing.T) {
	for name, data := range map[string][]byte{
		"text":  []byte(strings.Repeat("hello", 50)),
		"html":  []byte("<html><script>alert(1)</script></html>"),
		"gif":   []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"empty": {},
	} {
		if _, err := processImage(data); err != ErrUnsupportedType {
			t.Fatalf("%s: expected ErrUnsupportedType, got %v", name, err)
		}
	}
}

func TestProcessImageRejectsDecompressionBomb(t *testing.T) {
	// A tiny PNG whose header claims 20000x20000 pixels.
	img := image.NewGray(image.Rect(0, 0, 20000, 20000))
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if buf.Len() > 2<<20 {
		t.Skip("encoded test image unexpectedly large")
	}
	if _, err := processImage(buf.Bytes()); err != ErrImageTooLarge {
		t.Fatalf("expected ErrImageTooLarge, got %v", err)
	}
}

func TestProcessImageStripsTrailingPayloadAndResizes(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2000, 500))
	for x := 0; x < 2000; x += 7 {
		img.Set(x, 10, color.White)
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	payload := []byte("SECRET-PAYLOAD-APPENDED")
	out, err := processImage(append(buf.Bytes(), payload...))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.data, payload) {
		t.Fatal("trailing payload must not survive re-encoding")
	}
	if out.width != 1280 || out.height != 320 {
		t.Fatalf("unexpected size %dx%d", out.width, out.height)
	}
}

func TestLocalStorageRejectsPathTraversal(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../etc/passwd", "a.jpg", "/etc/passwd", "11111111-1111-1111-1111-111111111111.jpg/../x"} {
		if err := s.Put(nil, key, []byte("x")); err == nil { //nolint:staticcheck
			t.Fatalf("key %q must be rejected", key)
		}
	}
}
