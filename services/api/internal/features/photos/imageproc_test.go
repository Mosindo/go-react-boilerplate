package photos

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
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessImageReencodesAndDownscales(t *testing.T) {
	out, w, h, err := processImage(encodePNG(t, 2000, 1000))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if w != 1440 || h != 720 {
		t.Fatalf("unexpected size %dx%d", w, h)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("output is not a JPEG: %v", err)
	}
}

func TestProcessImageRejectsInvalidInput(t *testing.T) {
	if _, _, _, err := processImage([]byte("<?php echo 'hi'; ?>")); err == nil {
		t.Fatal("expected non-image to be rejected")
	}
	if _, _, _, err := processImage(encodePNG(t, 50, 50)); err == nil {
		t.Fatal("expected tiny image to be rejected")
	}
}

func TestProcessImageRejectsDecompressionBombs(t *testing.T) {
	img := image.NewNRGBA64(image.Rect(0, 0, 6320, 6320))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := processImage(buf.Bytes()); err != errImageTooLarge {
		t.Fatalf("expected errImageTooLarge for a 16-bit 40MP image, got %v", err)
	}
}

func TestApplyOrientationRotates(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	rotated := applyOrientation(img, 6)
	if rotated.Bounds().Dx() != 2 || rotated.Bounds().Dy() != 4 {
		t.Fatalf("unexpected rotated bounds %v", rotated.Bounds())
	}
	if r, _, _, _ := rotated.At(1, 0).RGBA(); r>>8 != 255 {
		t.Fatal("top-left pixel should move to top-right for orientation 6")
	}
}

func TestJPEGOrientationParsesExif(t *testing.T) {
	// Minimal JPEG prefix with an APP1/Exif segment holding orientation=6 (big endian).
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 6, 0, 0, 0, 0, 0, 0}
	segment := append([]byte("Exif\x00\x00"), tiff...)
	size := len(segment) + 2
	data := []byte{0xFF, 0xD8, 0xFF, 0xE1, byte(size >> 8), byte(size)}
	data = append(data, segment...)
	data = append(data, 0xFF, 0xD9)
	if got := jpegOrientation(data); got != 6 {
		t.Fatalf("expected orientation 6, got %d", got)
	}
}
