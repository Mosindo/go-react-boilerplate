package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
	"time"
)

func fixture(w, h int, asPNG bool) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 90, A: 128}) // semi transparent
		}
	}
	var buf bytes.Buffer
	if asPNG {
		_ = png.Encode(&buf, img)
	} else {
		_ = jpeg.Encode(&buf, img, nil)
	}
	return buf.Bytes()
}

func TestProcessImageDownscalesAndReencodesToJPEG(t *testing.T) {
	for _, asPNG := range []bool{true, false} {
		out, err := processImage(fixture(2400, 1800, asPNG))
		if err != nil {
			t.Fatalf("png=%v: %v", asPNG, err)
		}
		if out.Width != fullMaxEdge || out.Height != 960 {
			t.Fatalf("expected 1280x960, got %dx%d", out.Width, out.Height)
		}
		for name, data := range map[string][]byte{"full": out.Full, "thumb": out.Thumb} {
			img, format, err := image.Decode(bytes.NewReader(data))
			if err != nil || format != "jpeg" {
				t.Fatalf("%s must be a jpeg: %v %s", name, err, format)
			}
			if name == "thumb" && (img.Bounds().Dx() > thumbMaxEdge || img.Bounds().Dy() > thumbMaxEdge) {
				t.Fatalf("thumb too large: %v", img.Bounds())
			}
		}
	}
}

func TestProcessImageStripsMetadata(t *testing.T) {
	// A JPEG with an EXIF/APP1 segment carrying a marker string.
	base := fixture(400, 400, false)
	marker := []byte("GPSLatitude-SECRET")
	seg := append([]byte{0xFF, 0xE1, 0x00, byte(len(marker) + 2)}, marker...)
	withExif := append(append([]byte{}, base[:2]...), append(seg, base[2:]...)...)

	out, err := processImage(withExif)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Full, marker) || bytes.Contains(out.Thumb, marker) {
		t.Fatal("metadata must not survive re-encoding")
	}
}

func TestProcessImageRejectsUnsafeInput(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want error
	}{
		"text":        {[]byte(strings.Repeat("hello world ", 100)), ErrUnsupportedImage},
		"php":         {[]byte("<?php echo 1; ?>" + strings.Repeat(" ", 600)), ErrUnsupportedImage},
		"gif":         {append([]byte("GIF89a"), make([]byte, 600)...), ErrUnsupportedImage},
		"svg":         {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), ErrUnsupportedImage},
		"truncated":   {fixture(400, 400, true)[:100], ErrUnsupportedImage},
		"too small":   {fixture(100, 100, true), ErrImageTooSmall},
		"empty":       {nil, ErrUnsupportedImage},
		"over 8 MB":   {make([]byte, maxUploadBytes+1), ErrFileTooLarge},
		"png header+": {append(fixture(300, 300, true)[:40], make([]byte, 700)...), ErrUnsupportedImage},
	}
	for name, tc := range cases {
		if _, err := processImage(tc.data); !errors.Is(err, tc.want) {
			t.Errorf("%s: want %v, got %v", name, tc.want, err)
		}
	}
}

func TestProcessImageRejectsDecompressionBomb(t *testing.T) {
	// A header declaring 20000x20000 pixels must be refused before any pixel decode.
	var buf bytes.Buffer
	_ = png.Encode(&buf, image.NewGray(image.Rect(0, 0, 300, 300)))
	data := buf.Bytes()
	// IHDR width/height live at bytes 16..23.
	copy(data[16:20], []byte{0x00, 0x00, 0x4E, 0x20})
	copy(data[20:24], []byte{0x00, 0x00, 0x4E, 0x20})
	if _, err := processImage(data); !errors.Is(err, ErrImageTooLarge) && !errors.Is(err, ErrUnsupportedImage) {
		t.Fatalf("bomb must be rejected, got %v", err)
	}
}

func TestSignerRoundTripAndTamperDetection(t *testing.T) {
	s := NewSigner([]byte("0123456789abcdef0123456789abcdef"))
	now := time.Unix(1_700_000_000, 0)
	raw := s.URL("photo-1", variantFull, now)

	parse := func(u string) (exp, sig string) {
		q := u[strings.Index(u, "?")+1:]
		for _, kv := range strings.Split(q, "&") {
			p := strings.SplitN(kv, "=", 2)
			if p[0] == "exp" {
				exp = p[1]
			} else if p[0] == "sig" {
				sig = p[1]
			}
		}
		return
	}
	exp, sig := parse(raw)
	if !s.Verify("photo-1", variantFull, exp, sig, now) {
		t.Fatal("fresh url must verify")
	}
	if s.Verify("photo-2", variantFull, exp, sig, now) {
		t.Fatal("signature is bound to the photo id")
	}
	if s.Verify("photo-1", variantThumb, exp, sig, now) {
		t.Fatal("signature is bound to the variant")
	}
	if s.Verify("photo-1", variantFull, exp, sig, now.Add(3*time.Hour)) {
		t.Fatal("expired url must not verify")
	}
	if s.Verify("photo-1", variantFull, "9999999999", sig, now) {
		t.Fatal("extending the expiry must break the signature")
	}
	other := NewSigner([]byte("another-secret-another-secret-12"))
	if other.Verify("photo-1", variantFull, exp, sig, now) {
		t.Fatal("a different secret must not verify")
	}
	if a, b := s.URL("photo-1", variantFull, now), s.URL("photo-1", variantFull, now.Add(10*time.Minute)); a != b && now.Unix()/3600 == now.Add(10*time.Minute).Unix()/3600 {
		t.Fatal("urls must be stable within the hour so caches work")
	}
}
