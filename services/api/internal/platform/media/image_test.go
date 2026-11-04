package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		img.Set(x, 0, color.RGBA{255, 0, 0, 255})
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func TestProcessResizesAndReencodesAsJPEG(t *testing.T) {
	out, err := Process(bytes.NewReader(pngBytes(2000, 1000)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1280 || out.Height != 640 {
		t.Fatalf("got %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output is not a jpeg: %v", err)
	}
}

func TestProcessRejectsNonImages(t *testing.T) {
	for name, data := range map[string][]byte{
		"text":   []byte("<?php system($_GET['c']); ?>"),
		"script": []byte("GIF89a<script>alert(1)</script>"),
		"empty":  {},
	} {
		if _, err := Process(bytes.NewReader(data)); err != ErrNotImage {
			t.Errorf("%s: expected ErrNotImage, got %v", name, err)
		}
	}
}

func TestProcessRejectsTinyAndHugeInputs(t *testing.T) {
	if _, err := Process(bytes.NewReader(pngBytes(50, 50))); err != ErrTooSmall {
		t.Fatalf("expected ErrTooSmall, got %v", err)
	}
	if _, err := Process(strings.NewReader(strings.Repeat("a", MaxUploadBytes+10))); err != ErrTooLarge {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
}

func TestSignerVerify(t *testing.T) {
	s := NewSigner([]byte("secret-secret-secret-secret-1234"))
	u := s.URL("abc")
	var exp, sig string
	for _, part := range strings.Split(strings.SplitN(u, "?", 2)[1], "&") {
		kv := strings.SplitN(part, "=", 2)
		if kv[0] == "exp" {
			exp = kv[1]
		} else {
			sig = kv[1]
		}
	}
	if !s.Verify("abc", exp, sig) {
		t.Fatal("valid signature rejected")
	}
	if s.Verify("other", exp, sig) || s.Verify("abc", exp, "00") {
		t.Fatal("tampered signature accepted")
	}
	if s.Verify("abc", "1", sig) {
		t.Fatal("expired url accepted")
	}
}

func TestStoreRejectsTraversal(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Path("../../etc/passwd"); err == nil {
		t.Fatal("traversal accepted")
	}
	key, err := s.Put([]byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(key); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(key); err != nil {
		t.Fatalf("double delete should be fine: %v", err)
	}
}
