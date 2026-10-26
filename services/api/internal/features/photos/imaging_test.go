package photos

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
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

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestProcessImageResizesAndReencodes(t *testing.T) {
	out, err := processImage(encodeJPEG(t, solid(3000, 2000, color.RGBA{200, 50, 50, 255})))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 1080 || out.Height != 720 {
		t.Fatalf("expected 1080x720, got %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be a valid jpeg: %v", err)
	}
}

func TestProcessImageAcceptsPNGWithAlpha(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, solid(400, 400, color.NRGBA{0, 0, 0, 0})); err != nil {
		t.Fatal(err)
	}
	if _, err := processImage(b.Bytes()); err != nil {
		t.Fatalf("png should be accepted: %v", err)
	}
}

func TestProcessImageRejectsDangerousInputs(t *testing.T) {
	var g bytes.Buffer
	_ = gif.Encode(&g, solid(300, 300, color.White), nil)

	cases := map[string][]byte{
		"gif":            g.Bytes(),
		"text":           []byte("<?php system($_GET['c']); ?>"),
		"html":           []byte("<html><script>alert(1)</script></html>"),
		"empty":          {},
		"truncated jpeg": encodeJPEG(t, solid(300, 300, color.White))[:100],
		"tiny":           encodeJPEG(t, solid(50, 50, color.White)),
	}
	for name, data := range cases {
		if _, err := processImage(data); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}

func TestProcessImageRejectsDecompressionBomb(t *testing.T) {
	// A PNG header declaring 30000x30000 pixels must be refused before decoding.
	var b bytes.Buffer
	if err := png.Encode(&b, solid(300, 300, color.White)); err != nil {
		t.Fatal(err)
	}
	data := b.Bytes()
	binary.BigEndian.PutUint32(data[16:20], 30000)
	binary.BigEndian.PutUint32(data[20:24], 30000)
	if _, err := processImage(data); err == nil {
		t.Fatal("oversized dimensions must be rejected")
	}
}

func exifJPEG(t *testing.T, img image.Image, orientation uint16) []byte {
	t.Helper()
	base := encodeJPEG(t, img)
	var seg bytes.Buffer
	seg.WriteString("Exif\x00\x00")
	seg.WriteString("MM")
	_ = binary.Write(&seg, binary.BigEndian, uint16(42))
	_ = binary.Write(&seg, binary.BigEndian, uint32(8))
	_ = binary.Write(&seg, binary.BigEndian, uint16(1))
	_ = binary.Write(&seg, binary.BigEndian, uint16(0x0112))
	_ = binary.Write(&seg, binary.BigEndian, uint16(3))
	_ = binary.Write(&seg, binary.BigEndian, uint32(1))
	_ = binary.Write(&seg, binary.BigEndian, orientation)
	_ = binary.Write(&seg, binary.BigEndian, uint16(0))
	_ = binary.Write(&seg, binary.BigEndian, uint32(0))
	out := []byte{0xFF, 0xD8, 0xFF, 0xE1}
	out = binary.BigEndian.AppendUint16(out, uint16(seg.Len()+2))
	out = append(out, seg.Bytes()...)
	return append(out, base[2:]...)
}

func TestProcessImageAppliesExifOrientation(t *testing.T) {
	// Landscape source flagged as "rotate 90° CW" must come out portrait.
	out, err := processImage(exifJPEG(t, solid(600, 300, color.RGBA{10, 120, 200, 255}), 6))
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if out.Width != 300 || out.Height != 600 {
		t.Fatalf("expected portrait 300x600, got %dx%d", out.Width, out.Height)
	}
}
