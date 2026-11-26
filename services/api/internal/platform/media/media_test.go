package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
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

func TestProcessReencodesAndBoundsSize(t *testing.T) {
	out, err := Process(encodePNG(t, 2000, 1000))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != maxOutputEdge || out.Height != 720 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output is not a JPEG: %v", err)
	}
}

func TestProcessRejectsDangerousInput(t *testing.T) {
	cases := map[string][]byte{
		"html":      []byte("<html><script>alert(1)</script></html>"),
		"gif":       []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"truncated": encodePNG(t, 400, 400)[:200],
		"tiny":      encodePNG(t, 50, 50),
	}
	for name, data := range cases {
		if _, err := Process(data); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestSignerVerify(t *testing.T) {
	signer := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	fixed := time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC)
	signer.now = func() time.Time { return fixed }

	key := "0123456789abcdef0123456789abcdef.jpg"
	url := signer.URL(key)
	if url != signer.URL(key) {
		t.Fatal("URL must be stable within a bucket")
	}

	exp := fixed.Truncate(time.Hour).Add(2 * time.Hour).Unix()
	sig := signer.sign(key, itoa(exp))
	if !signer.Verify(key, itoa(exp), sig) {
		t.Fatal("expected valid signature")
	}
	if signer.Verify("ffffffffffffffffffffffffffffffff.jpg", itoa(exp), sig) {
		t.Fatal("signature must be bound to the key")
	}
	signer.now = func() time.Time { return fixed.Add(3 * time.Hour) }
	if signer.Verify(key, itoa(exp), sig) {
		t.Fatal("expired signature must be rejected")
	}
}

func TestValidKey(t *testing.T) {
	if !ValidKey("0123456789abcdef0123456789abcdef.jpg") {
		t.Fatal("expected valid key")
	}
	for _, k := range []string{"../etc/passwd", "abc.jpg", "0123456789abcdef0123456789abcdef.png", ""} {
		if ValidKey(k) {
			t.Errorf("%q should be invalid", k)
		}
	}
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

func TestApplyOrientationSwapsDimensions(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	out := applyOrientation(src, 6) // rotate 90° clockwise
	if b := out.Bounds(); b.Dx() != 2 || b.Dy() != 4 {
		t.Fatalf("unexpected bounds %v", b)
	}
	if r, _, _, _ := out.At(1, 0).RGBA(); r>>8 != 255 {
		t.Fatal("top-left pixel should move to top-right")
	}
}
