package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{200, 80, 120, 255})
		}
	}
	return img
}

func TestProcessResizesPNGToJPEG(t *testing.T) {
	var in bytes.Buffer
	if err := png.Encode(&in, solid(3200, 1600)); err != nil {
		t.Fatal(err)
	}
	res, err := Process(&in)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if res.Width != 1600 || res.Height != 800 {
		t.Fatalf("unexpected size %dx%d", res.Width, res.Height)
	}
	if _, format, err := image.DecodeConfig(bytes.NewReader(res.Data)); err != nil || format != "jpeg" {
		t.Fatalf("output must be jpeg, got %q err=%v", format, err)
	}
}

func TestProcessRejectsNonImagesAndOtherFormats(t *testing.T) {
	if _, err := Process(bytes.NewReader([]byte("<?php system($_GET['c']); ?>"))); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected unsupported type, got %v", err)
	}
	var g bytes.Buffer
	if err := gif.Encode(&g, image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{color.Black, color.White}), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Process(&g); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("gif should be rejected, got %v", err)
	}
}

func TestProcessRejectsOversizedUpload(t *testing.T) {
	big := bytes.Repeat([]byte{0}, MaxInputBytes+10)
	if _, err := Process(bytes.NewReader(big)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
}

func TestApplyOrientationRotates(t *testing.T) {
	rotated := applyOrientation(solid(40, 20), 6)
	if b := rotated.Bounds(); b.Dx() != 20 || b.Dy() != 40 {
		t.Fatalf("orientation 6 should swap dimensions, got %v", b)
	}
}

func TestProcessKeepsSmallJPEG(t *testing.T) {
	var in bytes.Buffer
	if err := jpeg.Encode(&in, solid(64, 48), nil); err != nil {
		t.Fatal(err)
	}
	res, err := Process(&in)
	if err != nil || res.Width != 64 || res.Height != 48 {
		t.Fatalf("unexpected result %+v err=%v", res, err)
	}
}
