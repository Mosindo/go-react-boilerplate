package photos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	apperr "example.com/api/internal/platform/errors"
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

// jpegWithOrientation builds a JPEG carrying an EXIF APP1 segment with the
// orientation tag and a fake GPS marker string.
func jpegWithOrientation(t *testing.T, w, h int, orientation uint16) []byte {
	t.Helper()
	var plain bytes.Buffer
	if err := jpeg.Encode(&plain, image.NewRGBA(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatal(err)
	}
	tiff := new(bytes.Buffer)
	tiff.WriteString("II")
	binary.Write(tiff, binary.LittleEndian, uint16(42))
	binary.Write(tiff, binary.LittleEndian, uint32(8))
	binary.Write(tiff, binary.LittleEndian, uint16(1))
	binary.Write(tiff, binary.LittleEndian, uint16(0x0112))
	binary.Write(tiff, binary.LittleEndian, uint16(3))
	binary.Write(tiff, binary.LittleEndian, uint32(1))
	binary.Write(tiff, binary.LittleEndian, orientation)
	binary.Write(tiff, binary.LittleEndian, uint16(0))
	binary.Write(tiff, binary.LittleEndian, uint32(0))
	tiff.WriteString("GPSSECRET48.8566")

	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	segment := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(segment[2:], uint16(len(payload)+2))
	segment = append(segment, payload...)

	src := plain.Bytes()
	out := append([]byte{}, src[:2]...)
	out = append(out, segment...)
	return append(out, src[2:]...)
}

func TestProcessImageAcceptsPNGAndReencodesJPEG(t *testing.T) {
	out, err := ProcessImage(encodePNG(t, 400, 300))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 400 || out.Height != 300 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output is not a JPEG: %v", err)
	}
}

func TestProcessImageDownscalesLargeImages(t *testing.T) {
	out, err := ProcessImage(encodePNG(t, 3000, 1500))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != maxOutputSide || out.Height != 720 {
		t.Fatalf("expected 1440x720, got %dx%d", out.Width, out.Height)
	}
}

func TestProcessImageAppliesOrientationAndStripsMetadata(t *testing.T) {
	raw := jpegWithOrientation(t, 400, 240, 6)
	if got := jpegOrientation(raw); got != 6 {
		t.Fatalf("expected orientation 6, got %d", got)
	}
	out, err := ProcessImage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 240 || out.Height != 400 {
		t.Fatalf("expected rotated 240x400, got %dx%d", out.Width, out.Height)
	}
	if bytes.Contains(out.Data, []byte("GPSSECRET")) || bytes.Contains(out.Data, []byte("Exif")) {
		t.Fatal("metadata must be stripped from processed images")
	}
}

func TestProcessImageRejectsDangerousInput(t *testing.T) {
	cases := map[string][]byte{
		"html":      []byte("<html><script>alert(1)</script></html>"),
		"svg":       []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"empty":     {},
		"truncated": encodePNG(t, 400, 400)[:120],
	}
	for name, raw := range cases {
		if _, err := ProcessImage(raw); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	if _, err := ProcessImage(encodePNG(t, 100, 100)); !errors.Is(err, ErrImageTooSmall) {
		t.Fatalf("expected ErrImageTooSmall, got %v", err)
	}
	var appErr *apperr.AppError
	if _, err := ProcessImage([]byte("GIF89a....")); !errors.As(err, &appErr) || appErr.Code != "unsupported_media_type" {
		t.Fatalf("expected unsupported media type, got %v", err)
	}
}
