package media

import (
	"bytes"
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
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessResizesAndReencodes(t *testing.T) {
	out, err := Process(encodePNG(t, 3000, 1500))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1440 || out.Height != 720 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be a valid jpeg: %v", err)
	}
}

func TestProcessRejectsInvalidInputs(t *testing.T) {
	if _, err := Process([]byte("<?php echo 'hi'; ?>")); err != ErrUnsupportedType {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
	if _, err := Process(encodePNG(t, 100, 100)); err != ErrImageTooSmall {
		t.Fatalf("expected ErrImageTooSmall, got %v", err)
	}
	corrupt := encodePNG(t, 400, 400)[:120]
	if _, err := Process(corrupt); err != ErrCorruptImage {
		t.Fatalf("expected ErrCorruptImage, got %v", err)
	}
	if _, err := Process(nil); err != ErrCorruptImage {
		t.Fatalf("expected ErrCorruptImage for empty input, got %v", err)
	}
}

func TestJPEGOrientationParsing(t *testing.T) {
	// Minimal JPEG: SOI + APP1 Exif (little endian, 1 IFD entry orientation=6) + EOI.
	tiff := []byte{'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, 0x01, 0x00,
		0x12, 0x01, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	size := len(payload) + 2
	data := []byte{0xFF, 0xD8, 0xFF, 0xE1, byte(size >> 8), byte(size)}
	data = append(data, payload...)
	data = append(data, 0xFF, 0xD9)
	if got := jpegOrientation(data); got != 6 {
		t.Fatalf("expected orientation 6, got %d", got)
	}
	if got := jpegOrientation([]byte{0xFF, 0xD8, 0xFF, 0xD9}); got != 1 {
		t.Fatalf("expected default orientation, got %d", got)
	}
}

func TestApplyOrientationRotatesDimensions(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	rotated := applyOrientation(src, 6)
	if rotated.Bounds().Dx() != 2 || rotated.Bounds().Dy() != 4 {
		t.Fatalf("unexpected bounds %v", rotated.Bounds())
	}
	// Orientation 6 = rotate 90° clockwise: top-left pixel moves to top-right.
	if r, _, _, _ := rotated.At(1, 0).RGBA(); r == 0 {
		t.Fatal("expected red pixel at top-right after rotation")
	}
}
