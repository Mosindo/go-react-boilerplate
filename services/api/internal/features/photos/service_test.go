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
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{200, 50, 50, 128}) // semi-transparent
		}
	}
	return img
}

func TestProcessDownscalesAndReencodes(t *testing.T) {
	var buf bytes.Buffer
	_ = png.Encode(&buf, solid(3000, 1500))
	out, w, h, err := Process(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if w != 1280 || h != 640 {
		t.Fatalf("unexpected size %dx%d", w, h)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("output must be a valid jpeg: %v", err)
	}
}

func TestProcessRejects(t *testing.T) {
	var small bytes.Buffer
	_ = png.Encode(&small, solid(100, 100))
	cases := map[string][]byte{
		"empty":    nil,
		"text":     []byte("hello world"),
		"html":     []byte("<html><script>alert(1)</script></html>"),
		"too tiny": small.Bytes(),
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
}

func TestSignedURLsExpireAndBind(t *testing.T) {
	s := NewService(nil, nil, []byte("k"))
	exp := s.expiry()
	sig := s.sign("photo-1", exp)
	if sig == s.sign("photo-2", exp) || sig == s.sign("photo-1", exp+1) {
		t.Fatal("signature must bind id and expiry")
	}
}
