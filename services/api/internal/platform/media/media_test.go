package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"
)

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessUploadReencodesAndResizes(t *testing.T) {
	out, err := ProcessUpload(bytes.NewReader(testPNG(t, 2000, 1000)))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != MaxDimension || out.Height != 800 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if !bytes.HasPrefix(out.Data, []byte{0xFF, 0xD8, 0xFF}) {
		t.Fatal("expected JPEG output")
	}
}

func TestProcessUploadRejectsBadInput(t *testing.T) {
	if _, err := ProcessUpload(strings.NewReader("<?php echo 1; ?>")); err != ErrUnsupportedType {
		t.Fatalf("expected unsupported type, got %v", err)
	}
	if _, err := ProcessUpload(bytes.NewReader(testPNG(t, 50, 50))); err != ErrImageTooSmall {
		t.Fatalf("expected too small, got %v", err)
	}
	// A PNG header followed by garbage must not decode.
	broken := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)
	if _, err := ProcessUpload(bytes.NewReader(broken)); err != ErrInvalidImage {
		t.Fatalf("expected invalid image, got %v", err)
	}
	huge := bytes.Repeat([]byte{0}, MaxUploadBytes+10)
	if _, err := ProcessUpload(bytes.NewReader(huge)); err != ErrImageTooLarge {
		t.Fatalf("expected too large, got %v", err)
	}
}

func TestSignerRoundTripAndExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	s := NewSigner("0123456789abcdef0123456789abcdef")
	s.now = func() time.Time { return now }
	url := s.PhotoURL("photo-1")
	parts := strings.Split(strings.SplitN(url, "?", 2)[1], "&")
	exp := strings.TrimPrefix(parts[0], "exp=")
	sig := strings.TrimPrefix(parts[1], "sig=")
	if !s.Verify("photo-1", exp, sig) {
		t.Fatal("expected signature to verify")
	}
	if s.Verify("photo-2", exp, sig) {
		t.Fatal("signature must be bound to the photo id")
	}
	if url != s.PhotoURL("photo-1") {
		t.Fatal("URL should be stable within the same hour")
	}
	now = now.Add(3 * time.Hour)
	if s.Verify("photo-1", exp, sig) {
		t.Fatal("expired signature must be rejected")
	}
}

func TestDiskStorageRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	s, err := NewDiskStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Path("../../etc/passwd"); err != ErrInvalidKey {
		t.Fatalf("expected invalid key, got %v", err)
	}
	key, _ := NewKey()
	if err := s.Save(key, strings.NewReader("data")); err != nil {
		t.Fatal(err)
	}
	f, err := s.Open(key)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := s.Delete(key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(key); err != ErrNotFound {
		t.Fatalf("expected not found after delete, got %v", err)
	}
	_ = os.RemoveAll(dir)
}
