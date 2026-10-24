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

func pngBytes(w, h int) []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h)))
	return b.Bytes()
}

func TestProcessImageResizesAndReencodes(t *testing.T) {
	out, err := processImage(pngBytes(2000, 1000))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1080 || out.Height != 540 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	full, err := jpeg.Decode(bytes.NewReader(out.Full))
	if err != nil || full.Bounds().Dx() != 1080 {
		t.Fatalf("full must be a 1080px jpeg: %v", err)
	}
	thumb, _ := jpeg.Decode(bytes.NewReader(out.Thumb))
	if thumb.Bounds().Dx() != 320 {
		t.Fatalf("thumb width %d", thumb.Bounds().Dx())
	}
}

func TestProcessImageDoesNotUpscale(t *testing.T) {
	out, err := processImage(pngBytes(100, 50))
	if err != nil || out.Width != 100 || out.Height != 50 {
		t.Fatalf("small images keep their size: %+v %v", out, err)
	}
}

func TestProcessImageRejects(t *testing.T) {
	if _, err := processImage([]byte("not an image")); !errors.Is(err, ErrUnsupportedImage) {
		t.Fatalf("garbage: %v", err)
	}
	if _, err := processImage(pngBytes(10001, 1)); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("too wide: %v", err)
	}
	if _, err := processImage(pngBytes(6000, 5000)); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("too many pixels: %v", err)
	}
	if _, err := processImage(pngBytes(100, 100)[:60]); err == nil {
		t.Fatal("truncated image must fail")
	}
}

func TestApplyOrientation(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1)) // [red][blue]
	red, blue := color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}
	src.Set(0, 0, red)
	src.Set(1, 0, blue)

	flipped := applyOrientation(src, 2)
	if flipped.At(0, 0) != blue || flipped.At(1, 0) != red {
		t.Fatal("orientation 2 should mirror horizontally")
	}
	rot := applyOrientation(src, 6) // 90° clockwise: 1x2, red on top
	if rot.Bounds().Dx() != 1 || rot.Bounds().Dy() != 2 || rot.At(0, 0) != red || rot.At(0, 1) != blue {
		t.Fatal("orientation 6 should rotate 90° clockwise")
	}
	if applyOrientation(src, 1) != image.Image(src) {
		t.Fatal("orientation 1 is a no-op")
	}
}

func TestExifOrientation(t *testing.T) {
	// minimal JPEG prefix: SOI + APP1(Exif, big endian, orientation=6)
	exif := []byte{
		0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x22,
		'E', 'x', 'i', 'f', 0, 0,
		'M', 'M', 0x00, 0x2A, 0x00, 0x00, 0x00, 0x08,
		0x00, 0x01, // one entry
		0x01, 0x12, 0x00, 0x03, 0x00, 0x00, 0x00, 0x01, 0x00, 0x06, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	if got := exifOrientation(exif); got != 6 {
		t.Fatalf("expected orientation 6, got %d", got)
	}
	if exifOrientation([]byte("nope")) != 1 || exifOrientation(nil) != 1 {
		t.Fatal("missing EXIF defaults to 1")
	}
}
