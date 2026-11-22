package media

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"strconv"
	"testing"
	"time"
)

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessUploadReencodesAndDownscales(t *testing.T) {
	out, err := ProcessUpload(encodePNG(t, 3200, 1600))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1600 || out.Height != 800 {
		t.Fatalf("expected 1600x800, got %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.DecodeConfig(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be a JPEG: %v", err)
	}
}

func TestProcessUploadRejectsDangerousInput(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want error
	}{
		"script disguised": {[]byte("<script>alert(1)</script>"), ErrUnsupportedType},
		"svg":              {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), ErrUnsupportedType},
		"too small":        {encodePNG(t, 50, 50), ErrImageTooSmall},
		"empty":            {nil, ErrCorruptImage},
		"truncated png":    {encodePNG(t, 400, 400)[:200], ErrCorruptImage},
	}
	for name, tc := range cases {
		if _, err := ProcessUpload(tc.data); !errors.Is(err, tc.want) {
			t.Errorf("%s: expected %v, got %v", name, tc.want, err)
		}
	}
}

func TestSignerRoundTrip(t *testing.T) {
	s := NewSigner([]byte("a-very-long-secret-used-only-for-tests"), time.Hour)
	now := time.Unix(1_800_000_000, 0)
	exp := s.Expiry(now)
	sig := s.sign("photo-1", exp)
	if _, ok := s.Verify("photo-1", itoa(exp), sig, now); !ok {
		t.Fatalf("valid signature rejected")
	}
	if _, ok := s.Verify("photo-2", itoa(exp), sig, now); ok {
		t.Fatalf("signature must be bound to the photo id")
	}
	if _, ok := s.Verify("photo-1", itoa(exp), sig, time.Unix(exp+1, 0)); ok {
		t.Fatalf("expired signature accepted")
	}
	if s.PhotoURL("p", now) != s.PhotoURL("p", now.Add(10*time.Minute)) {
		t.Fatalf("URLs should be stable inside a window for caching")
	}
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}

func TestLocalStorageRejectsTraversal(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"../etc/passwd", "a/b.jpg", "x.png", ".hidden.jpg"} {
		if err := s.Save(key, []byte("x")); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("key %q should be rejected, got %v", key, err)
		}
	}
	if err := s.Save("abc-123.jpg", []byte("data")); err != nil {
		t.Fatalf("save: %v", err)
	}
	f, err := s.Open("abc-123.jpg")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	got, _ := io.ReadAll(f)
	f.Close()
	if string(got) != "data" {
		t.Fatalf("unexpected content %q", got)
	}
	if err := s.Delete("abc-123.jpg"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.Open("abc-123.jpg"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}
