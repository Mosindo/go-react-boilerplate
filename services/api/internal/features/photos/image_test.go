package photos

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func mustJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// withOrientation injects an EXIF APP1 segment carrying the orientation tag after the SOI marker.
func withOrientation(jpegData []byte, orientation uint16) []byte {
	var tiff bytes.Buffer
	tiff.WriteString("MM")
	_ = binary.Write(&tiff, binary.BigEndian, uint16(42))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(8))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(1))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(0x0112))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(3))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(1))
	_ = binary.Write(&tiff, binary.BigEndian, orientation)
	_ = binary.Write(&tiff, binary.BigEndian, uint16(0))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(0))
	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	seg := []byte{0xFF, 0xE1}
	seg = binary.BigEndian.AppendUint16(seg, uint16(len(payload)+2))
	seg = append(seg, payload...)
	out := append([]byte{}, jpegData[:2]...)
	out = append(out, seg...)
	return append(out, jpegData[2:]...)
}

func decodeSize(t *testing.T, data []byte) (int, int) {
	t.Helper()
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "jpeg" {
		t.Fatalf("output must be a JPEG: %v %s", err, format)
	}
	return cfg.Width, cfg.Height
}

func TestProcessImageDownscalesAndProducesThumb(t *testing.T) {
	out, err := processImage(mustJPEG(t, solid(3000, 2000, color.RGBA{200, 50, 50, 255})))
	if err != nil {
		t.Fatal(err)
	}
	if w, h := decodeSize(t, out.Full); w != 1280 || h != 853 {
		t.Errorf("full size %dx%d", w, h)
	}
	if w, h := decodeSize(t, out.Thumb); w != 360 || h != 240 {
		t.Errorf("thumb size %dx%d", w, h)
	}
	if out.Width != 1280 {
		t.Errorf("reported width %d", out.Width)
	}
}

func TestProcessImageAppliesExifOrientation(t *testing.T) {
	// A landscape 600x300 file tagged "rotate 90 CW" must come out portrait 300x600.
	data := withOrientation(mustJPEG(t, solid(600, 300, color.White)), 6)
	out, err := processImage(data)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := decodeSize(t, out.Full); w != 300 || h != 600 {
		t.Fatalf("orientation not applied: %dx%d", w, h)
	}
}

func TestProcessImageStripsMetadata(t *testing.T) {
	data := withOrientation(mustJPEG(t, solid(400, 400, color.White)), 1)
	out, err := processImage(data)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Full, []byte("Exif")) {
		t.Fatal("EXIF must not survive re-encoding")
	}
}

func TestProcessImageAcceptsPNGWithAlpha(t *testing.T) {
	var b bytes.Buffer
	_ = png.Encode(&b, solid(400, 400, color.NRGBA{0, 0, 0, 0}))
	out, err := processImage(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	img, _ := jpeg.Decode(bytes.NewReader(out.Full))
	r, g, bl, _ := img.At(10, 10).RGBA()
	if r>>8 < 240 || g>>8 < 240 || bl>>8 < 240 {
		t.Fatal("transparent PNG should be flattened onto white")
	}
}

func TestProcessImageRejects(t *testing.T) {
	cases := map[string][]byte{
		"text":      []byte("hello world, not an image at all"),
		"php":       []byte("<?php system($_GET['c']); ?>"),
		"gif":       []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"svg":       []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`),
		"truncated": mustJPEG(t, solid(400, 400, color.White))[:300],
		"empty":     {},
	}
	for name, data := range cases {
		if _, err := processImage(data); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if _, err := processImage(mustJPEG(t, solid(150, 900, color.White))); err != errTooSmall {
		t.Errorf("expected errTooSmall, got %v", err)
	}
}

func TestJPEGOrientationMalformedIsSafe(t *testing.T) {
	for _, data := range [][]byte{nil, {0xFF}, {0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x04}, {0xFF, 0xD8, 0xFF, 0xE1, 0xFF, 0xFF, 'E'}} {
		if got := jpegOrientation(data); got != 1 {
			t.Errorf("malformed EXIF must default to 1, got %d", got)
		}
	}
}
