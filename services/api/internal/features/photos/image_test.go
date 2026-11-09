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

func testImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	return img
}

func encodeJPEG(t *testing.T, w, h int) []byte {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, testImage(w, h), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestProcessImageDownscalesAndReencodes(t *testing.T) {
	data, w, h, err := ProcessImage(encodeJPEG(t, 2000, 1000))
	if err != nil {
		t.Fatal(err)
	}
	if w != 1280 || h != 640 {
		t.Fatalf("unexpected size %dx%d", w, h)
	}
	if _, err := jpeg.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("output is not a valid jpeg: %v", err)
	}
}

func TestProcessImageAcceptsPNG(t *testing.T) {
	var b bytes.Buffer
	_ = png.Encode(&b, testImage(200, 200))
	if _, _, _, err := ProcessImage(b.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func TestProcessImageRejectsBadInput(t *testing.T) {
	cases := map[string][]byte{
		"empty":   {},
		"text":    []byte("<?php system($_GET['x']); ?>"),
		"html":    []byte("<html><script>alert(1)</script></html>"),
		"gif":     []byte("GIF89a\x01\x00\x01\x00"),
		"tiny":    encodeJPEG(t, 10, 10),
		"corrupt": append([]byte{0xff, 0xd8, 0xff, 0xe0}, bytes.Repeat([]byte{1}, 100)...),
	}
	for name, raw := range cases {
		if _, _, _, err := ProcessImage(raw); !errors.Is(err, ErrInvalidImage) {
			t.Errorf("%s: expected ErrInvalidImage, got %v", name, err)
		}
	}
}

func TestProcessImageDropsTrailingPayload(t *testing.T) {
	raw := append(encodeJPEG(t, 100, 100), []byte("SECRET-PAYLOAD")...)
	data, _, _, err := ProcessImage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("SECRET-PAYLOAD")) {
		t.Fatal("payload survived re-encoding")
	}
}

func TestDiskStorageRejectsTraversal(t *testing.T) {
	s, err := NewDiskStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save("../evil.jpg", []byte("x")); err == nil {
		t.Fatal("traversal key accepted")
	}
	key, _ := NewKey()
	if err := s.Save(key, []byte("ok")); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Read(key); string(got) != "ok" {
		t.Fatal("roundtrip failed")
	}
	if err := s.Delete(key); err != nil {
		t.Fatal(err)
	}
}
