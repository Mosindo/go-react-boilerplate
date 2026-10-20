package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{200, 50, 50, 255})
		}
	}
	return img
}

func TestApplyOrientationSwapsDimensions(t *testing.T) {
	src := solid(30, 10)
	for _, o := range []int{5, 6, 7, 8} {
		if b := applyOrientation(src, o).Bounds(); b.Dx() != 10 || b.Dy() != 30 {
			t.Errorf("orientation %d: got %dx%d", o, b.Dx(), b.Dy())
		}
	}
	for _, o := range []int{1, 2, 3, 4} {
		if b := applyOrientation(src, o).Bounds(); b.Dx() != 30 || b.Dy() != 10 {
			t.Errorf("orientation %d: got %dx%d", o, b.Dx(), b.Dy())
		}
	}
}

func TestExifOrientationParsing(t *testing.T) {
	// Minimal big-endian TIFF with one IFD entry: orientation = 6.
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 6, 0, 0, 0, 0, 0, 0}
	if got := exifOrientation(tiff); got != 6 {
		t.Fatalf("orientation = %d", got)
	}
	seg := append([]byte("Exif\x00\x00"), tiff...)
	jpg := []byte{0xFF, 0xD8, 0xFF, 0xE1, byte((len(seg) + 2) >> 8), byte(len(seg) + 2)}
	jpg = append(jpg, seg...)
	jpg = append(jpg, 0xFF, 0xD9)
	if got := jpegOrientation(jpg); got != 6 {
		t.Fatalf("jpeg orientation = %d", got)
	}
	if got := exifOrientation([]byte("garbage")); got != 1 {
		t.Fatalf("garbage must default to 1, got %d", got)
	}
}

func TestProcessImageDownsizesAndFlattens(t *testing.T) {
	var buf bytes.Buffer
	transparent := image.NewNRGBA(image.Rect(0, 0, 2000, 500)) // fully transparent PNG
	if err := png.Encode(&buf, transparent); err != nil {
		t.Fatal(err)
	}
	out, w, h, err := ProcessImage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if w != 1280 || h != 320 {
		t.Fatalf("size = %dx%d", w, h)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := decoded.At(10, 10).RGBA()
	if r>>8 < 250 || g>>8 < 250 || b>>8 < 250 {
		t.Fatalf("transparency should flatten to white, got %d %d %d", r>>8, g>>8, b>>8)
	}
}

func TestProcessImageRejectsNonImages(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":  {},
		"script": []byte("<?php system($_GET['c']); ?>"),
		"zip":    {'P', 'K', 3, 4, 0, 0, 0, 0},
	} {
		if _, _, _, err := ProcessImage(data); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
