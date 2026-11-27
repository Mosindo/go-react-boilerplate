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

func solidImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withExif inserts an APP1 segment carrying an orientation tag and a fake
// GPS marker string right after the SOI marker.
func withExif(jpegData []byte, orientation uint16) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1}
	entry := make([]byte, 12)
	binary.BigEndian.PutUint16(entry[0:2], 0x0112)
	binary.BigEndian.PutUint16(entry[2:4], 3)
	binary.BigEndian.PutUint32(entry[4:8], 1)
	binary.BigEndian.PutUint16(entry[8:10], orientation)
	tiff = append(tiff, entry...)
	tiff = append(tiff, 0, 0, 0, 0)
	tiff = append(tiff, []byte("GPS-SECRET-48.8566,2.3522")...)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:4], uint16(len(payload)+2))
	seg = append(seg, payload...)
	out := append([]byte{}, jpegData[:2]...)
	out = append(out, seg...)
	return append(out, jpegData[2:]...)
}

func TestProcessImageAppliesOrientationAndStripsMetadata(t *testing.T) {
	raw := withExif(encodeJPEG(t, solidImage(400, 300)), 6)
	if jpegOrientation(raw) != 6 {
		t.Fatalf("expected orientation 6, got %d", jpegOrientation(raw))
	}
	out, err := processImage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 300 || out.Height != 400 {
		t.Fatalf("expected rotated 300x400, got %dx%d", out.Width, out.Height)
	}
	if bytes.Contains(out.Data, []byte("GPS-SECRET")) || bytes.Contains(out.Data, []byte("Exif")) {
		t.Fatal("metadata must be stripped from the output")
	}
}

func TestProcessImageResizesLargeImages(t *testing.T) {
	out, err := processImage(encodeJPEG(t, solidImage(2400, 1200)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != OutputMaxSide || out.Height != 800 {
		t.Fatalf("expected %dx800, got %dx%d", OutputMaxSide, out.Width, out.Height)
	}
}

func TestProcessImageAcceptsPNG(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, solidImage(300, 300)); err != nil {
		t.Fatal(err)
	}
	out, err := processImage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out.Data, []byte{0xFF, 0xD8}) {
		t.Fatal("output must be JPEG")
	}
}

func TestProcessImageRejectsDangerousInput(t *testing.T) {
	if _, err := processImage([]byte("<?php system($_GET['c']); ?>")); err != ErrUnsupportedType {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
	if _, err := processImage([]byte("<svg xmlns='http://www.w3.org/2000/svg'><script>alert(1)</script></svg>")); err != ErrUnsupportedType {
		t.Fatalf("SVG must be rejected, got %v", err)
	}
	if _, err := processImage(encodeJPEG(t, solidImage(50, 50))); err != ErrImageTooSmall {
		t.Fatalf("expected ErrImageTooSmall, got %v", err)
	}

	// Decompression bomb: a valid PNG header announcing 30000x30000 pixels.
	var buf bytes.Buffer
	if err := png.Encode(&buf, solidImage(1, 1)); err != nil {
		t.Fatal(err)
	}
	bomb := buf.Bytes()
	binary.BigEndian.PutUint32(bomb[16:20], 30000)
	binary.BigEndian.PutUint32(bomb[20:24], 30000)
	binary.BigEndian.PutUint32(bomb[29:33], crc32.ChecksumIEEE(bomb[12:29]))
	if _, err := processImage(bomb); err != ErrImageTooLarge {
		t.Fatalf("expected ErrImageTooLarge, got %v", err)
	}

	truncated := encodeJPEG(t, solidImage(400, 400))[:200]
	if _, err := processImage(truncated); err == nil {
		t.Fatal("truncated image must be rejected")
	}
}
