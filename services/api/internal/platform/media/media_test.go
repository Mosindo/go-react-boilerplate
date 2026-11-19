package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"
)

func pngBytes(t *testing.T, w, h int) []byte {
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

func TestProcessResizesAndReencodes(t *testing.T) {
	out, err := Process(pngBytes(t, 2000, 1000))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1440 || out.Height != 720 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if !bytes.HasPrefix(out.Data, []byte{0xFF, 0xD8}) {
		t.Fatal("output must be JPEG")
	}
}

func TestProcessRejectsDangerousInput(t *testing.T) {
	cases := map[string][]byte{
		"html":   []byte("<html><script>alert(1)</script></html>"),
		"svg":    []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
		"empty":  {},
		"gif":    []byte("GIF89a\x01\x00\x01\x00"),
		"badpng": append([]byte("\x89PNG\r\n\x1a\n"), []byte("garbage")...),
	}
	for name, data := range cases {
		if _, err := Process(data); err == nil {
			t.Fatalf("%s: expected rejection", name)
		}
	}
	if _, err := Process(pngBytes(t, 100, 100)); err != ErrTooSmall {
		t.Fatalf("expected ErrTooSmall, got %v", err)
	}
}

func TestSignerRoundTrip(t *testing.T) {
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	now := time.Unix(1_700_000_000, 0)
	s.now = func() time.Time { return now }
	key := "0123456789abcdef0123456789abcdef.jpg"

	u := s.URL(key)
	if !strings.HasPrefix(u, "/media/"+key+"?") {
		t.Fatalf("unexpected url %s", u)
	}
	q := u[strings.Index(u, "?")+1:]
	var exp, sig string
	for _, part := range strings.Split(q, "&") {
		kv := strings.SplitN(part, "=", 2)
		if kv[0] == "exp" {
			exp = kv[1]
		}
		if kv[0] == "sig" {
			sig = kv[1]
		}
	}
	if !s.Verify(key, exp, sig) {
		t.Fatal("signature should verify")
	}
	if s.Verify("ffffffffffffffffffffffffffffffff.jpg", exp, sig) {
		t.Fatal("signature must be bound to the key")
	}
	now = now.Add(3 * time.Hour)
	if s.Verify(key, exp, sig) {
		t.Fatal("expired URL must be rejected")
	}
}

func TestValidKey(t *testing.T) {
	if !ValidKey("0123456789abcdef0123456789abcdef.jpg") {
		t.Fatal("expected valid key")
	}
	for _, k := range []string{"../etc/passwd", "abc.jpg", "0123456789abcdef0123456789abcdef.png", ""} {
		if ValidKey(k) {
			t.Fatalf("expected %q to be invalid", k)
		}
	}
}
