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

func encodePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestProcessImageReencodesAndResizes(t *testing.T) {
	out, err := ProcessImage(encodePNG(t, 2400, 1200))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1600 || out.Height != 800 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be a JPEG: %v", err)
	}
}

func TestProcessImageStripsMetadata(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	raw := buf.Bytes()
	// Inject a fake APP1/EXIF segment right after SOI.
	exif := append([]byte{0xFF, 0xE1, 0x00, 0x10}, []byte("Exif\x00\x00GPS-DATA")...)
	withExif := append(append([]byte{}, raw[:2]...), append(exif, raw[2:]...)...)

	out, err := ProcessImage(withExif)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if bytes.Contains(out.Data, []byte("GPS-DATA")) || bytes.Contains(out.Data, []byte("Exif")) {
		t.Fatal("metadata must be stripped")
	}
}

func TestProcessImageRejectsBadInput(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want error
	}{
		"text":      {[]byte("<html><script>alert(1)</script></html>"), ErrUnsupportedType},
		"gif":       {[]byte("GIF89a\x01\x00\x01\x00"), ErrUnsupportedType},
		"too small": {encodePNG(t, 100, 100), ErrImageTooSmall},
		"truncated": {encodePNG(t, 300, 300)[:120], ErrCorruptImage},
	}
	for name, tc := range cases {
		if _, err := ProcessImage(tc.data); !errors.Is(err, tc.want) {
			t.Errorf("%s: expected %v, got %v", name, tc.want, err)
		}
	}
}

func TestProcessImageRejectsDecompressionBomb(t *testing.T) {
	// A PNG header that claims 20000x20000 pixels: rejected before decoding.
	var buf bytes.Buffer
	img := image.NewGray(image.Rect(0, 0, 1, 1))
	_ = png.Encode(&buf, img)
	data := buf.Bytes()
	// IHDR width/height live at bytes 16..24.
	copy(data[16:20], []byte{0x00, 0x00, 0x4E, 0x20})
	copy(data[20:24], []byte{0x00, 0x00, 0x4E, 0x20})
	if _, err := ProcessImage(data); !errors.Is(err, ErrImageTooLarge) && !errors.Is(err, ErrCorruptImage) {
		t.Fatalf("expected bomb to be rejected, got %v", err)
	}
}
