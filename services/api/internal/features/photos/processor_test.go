package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{10, 120, 200, 255})
		}
	}
	return img
}

func TestProcessResizesAndReencodesToJPEG(t *testing.T) {
	var in bytes.Buffer
	_ = png.Encode(&in, solid(2400, 1200))
	out, w, h, err := Process(in.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if w != 1080 || h != 540 {
		t.Fatalf("unexpected size %dx%d", w, h)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("output must be a valid JPEG: %v", err)
	}
}

func TestProcessRejects(t *testing.T) {
	var small bytes.Buffer
	_ = png.Encode(&small, solid(100, 100))
	cases := map[string][]byte{
		"empty":   nil,
		"text":    []byte("hello world, definitely not an image"),
		"gif-ish": []byte("GIF89a....."),
		"small":   small.Bytes(),
		"truncated": func() []byte {
			var b bytes.Buffer
			_ = png.Encode(&b, solid(400, 400))
			return b.Bytes()[:200]
		}(),
	}
	for name, data := range cases {
		if _, _, _, err := Process(data); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
	if _, _, _, err := Process(make([]byte, MaxUploadBytes+1)); err != ErrTooLarge {
		t.Errorf("oversized: %v", err)
	}
}

func TestStorageKeyValidation(t *testing.T) {
	s, err := NewDiskStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Save("../../etc/passwd", []byte("x")); err == nil {
		t.Fatal("path traversal must be rejected")
	}
	key, _ := newKey()
	if err := s.Save(key, []byte("x")); err != nil {
		t.Fatal(err)
	}
	s.Remove([]string{key, "../evil"})
}
