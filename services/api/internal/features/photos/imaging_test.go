package photos

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solidPNG(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: 10, G: 120, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestProcessResizesAndReencodes(t *testing.T) {
	out, err := process(solidPNG(2400, 1200))
	if err != nil {
		t.Fatal(err)
	}
	full, err := jpeg.DecodeConfig(bytes.NewReader(out.Full))
	if err != nil {
		t.Fatalf("full is not a JPEG: %v", err)
	}
	if full.Width != 1080 || full.Height != 540 {
		t.Fatalf("full size %dx%d", full.Width, full.Height)
	}
	thumb, _ := jpeg.DecodeConfig(bytes.NewReader(out.Thumb))
	if thumb.Width != 320 || thumb.Height != 160 {
		t.Fatalf("thumb size %dx%d", thumb.Width, thumb.Height)
	}
}

func TestProcessKeepsSmallImages(t *testing.T) {
	out, err := process(solidPNG(100, 80))
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := jpeg.DecodeConfig(bytes.NewReader(out.Full))
	if cfg.Width != 100 || cfg.Height != 80 {
		t.Fatalf("small images must not be upscaled: %dx%d", cfg.Width, cfg.Height)
	}
}

func TestProcessRejectsNonImagesAndTrailingPayload(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":  nil,
		"text":   []byte("hello world, definitely not an image"),
		"html":   []byte("<html><script>alert(1)</script></html>"),
		"gif":    []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"broken": append([]byte("\x89PNG\r\n\x1a\n"), 1, 2, 3, 4),
	} {
		if _, err := process(data); err == nil {
			t.Fatalf("%s should be rejected", name)
		}
	}

	// Anything appended to a valid image is dropped by re-encoding.
	polyglot := append(solidPNG(10, 10), []byte("<?php echo 'pwned'; ?>")...)
	out, err := process(polyglot)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Full, []byte("<?php")) || bytes.Contains(out.Thumb, []byte("<?php")) {
		t.Fatal("trailing payload survived re-encoding")
	}
}

func TestProcessRejectsDecompressionBomb(t *testing.T) {
	// Minimal PNG header declaring 60000x60000 pixels.
	var buf bytes.Buffer
	buf.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], 60000)
	binary.BigEndian.PutUint32(ihdr[4:8], 60000)
	ihdr[8], ihdr[9] = 8, 2
	binary.Write(&buf, binary.BigEndian, uint32(13))
	buf.WriteString("IHDR")
	buf.Write(ihdr)
	crc := crc32IEEE(append([]byte("IHDR"), ihdr...))
	binary.Write(&buf, binary.BigEndian, crc)
	if _, err := process(buf.Bytes()); err == nil {
		t.Fatal("oversized canvas must be rejected before decoding")
	}
}

func TestEXIFOrientationIsApplied(t *testing.T) {
	// 4x2 image with a red left half. Orientation 6 (rotate 90 CW) must give a 2x4 image.
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			if x < 2 {
				src.Set(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				src.Set(x, y, color.RGBA{0, 0, 255, 255})
			}
		}
	}
	out := applyOrientation(src, 6)
	if b := out.Bounds(); b.Dx() != 2 || b.Dy() != 4 {
		t.Fatalf("rotated bounds %v", b)
	}
	// After a 90° clockwise rotation the red left half ends on top.
	if r, _, _, _ := out.At(0, 0).RGBA(); r>>8 != 255 {
		t.Fatal("top should be red after rotation")
	}
	if r, _, _, _ := out.At(0, 3).RGBA(); r>>8 != 0 {
		t.Fatal("bottom should be blue after rotation")
	}
	if applyOrientation(src, 1) != image.Image(src) {
		t.Fatal("orientation 1 is a no-op")
	}
}

func TestJPEGOrientationParsing(t *testing.T) {
	// SOI + APP1(Exif, big endian, one IFD0 entry: orientation = 8).
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 8, 0, 0, 0, 0}
	seg := append([]byte("Exif\x00\x00"), tiff...)
	data := []byte{0xFF, 0xD8, 0xFF, 0xE1}
	data = binary.BigEndian.AppendUint16(data, uint16(len(seg)+2))
	data = append(data, seg...)
	data = append(data, 0xFF, 0xDA)
	if got := jpegOrientation(data); got != 8 {
		t.Fatalf("orientation = %d", got)
	}
	if got := jpegOrientation([]byte("not a jpeg")); got != 1 {
		t.Fatalf("malformed input must default to 1, got %d", got)
	}
}

func crc32IEEE(b []byte) uint32 { return crc32.ChecksumIEEE(b) }
