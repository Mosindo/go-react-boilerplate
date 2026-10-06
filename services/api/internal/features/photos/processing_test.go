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
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withOrientation injects an EXIF APP1 segment carrying the orientation tag.
func withOrientation(jpegData []byte, orientation uint16) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8}
	entry := make([]byte, 2+12+4)
	binary.BigEndian.PutUint16(entry[0:], 1)
	binary.BigEndian.PutUint16(entry[2:], 0x0112)
	binary.BigEndian.PutUint16(entry[4:], 3)
	binary.BigEndian.PutUint32(entry[6:], 1)
	binary.BigEndian.PutUint16(entry[10:], orientation)
	payload := append([]byte("Exif\x00\x00"), append(tiff, entry...)...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	seg = append(seg, payload...)
	out := append([]byte{}, jpegData[:2]...)
	out = append(out, seg...)
	return append(out, jpegData[2:]...)
}

func TestProcessReencodesAndDownscales(t *testing.T) {
	out, err := Process(encodeJPEG(t, solid(3200, 2400, color.RGBA{200, 50, 50, 255})))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1600 || out.Height != 1200 {
		t.Fatalf("expected 1600x1200, got %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Fatalf("output must be a valid jpeg: %v", err)
	}
}

func TestProcessAcceptsPNGWithAlpha(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, solid(300, 300, color.NRGBA{0, 0, 0, 0})); err != nil {
		t.Fatal(err)
	}
	if _, err := Process(buf.Bytes()); err != nil {
		t.Fatalf("png should be accepted: %v", err)
	}
}

func TestProcessRejectsDangerousOrInvalidInput(t *testing.T) {
	cases := map[string][]byte{
		"empty":      nil,
		"html":       []byte("<html><script>alert(1)</script></html>"),
		"php":        []byte("<?php system($_GET['c']); ?>"),
		"truncated":  encodeJPEG(t, solid(300, 300, color.White))[:100],
		"tiny image": encodeJPEG(t, solid(50, 50, color.White)),
	}
	for name, data := range cases {
		if _, err := Process(data); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	// A polyglot with a JPEG header but a script payload after the image must
	// come out as pixels only.
	polyglot := append(encodeJPEG(t, solid(300, 300, color.White)), []byte("<?php evil() ?>")...)
	out, err := Process(polyglot)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Data, []byte("<?php")) {
		t.Fatal("re-encoding must drop trailing payloads")
	}
}

func TestProcessRejectsOversizedDimensions(t *testing.T) {
	var buf bytes.Buffer
	// 9000x20 passes the sniff but exceeds the side limit; declare it via a real PNG header.
	img := image.NewGray(image.Rect(0, 0, 9000, 20))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if _, err := Process(buf.Bytes()); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("expected ErrImageTooLarge, got %v", err)
	}
}

func TestProcessAppliesExifOrientation(t *testing.T) {
	// 400x200 landscape tagged "rotate 90 CW" must come out portrait 200x400.
	data := withOrientation(encodeJPEG(t, solid(400, 200, color.RGBA{10, 120, 200, 255})), 6)
	if o := jpegOrientation(data); o != 6 {
		t.Fatalf("orientation parse: got %d", o)
	}
	out, err := Process(data)
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 200 || out.Height != 400 {
		t.Fatalf("expected 200x400 after rotation, got %dx%d", out.Width, out.Height)
	}
}
