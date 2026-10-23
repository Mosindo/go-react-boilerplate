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

func testImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessResizesAndReencodesAsJPEG(t *testing.T) {
	out, err := Process(encodePNG(t, testImage(2000, 1000)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1080 || out.Height != 540 {
		t.Fatalf("unexpected full size %dx%d", out.Width, out.Height)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(out.Thumb))
	if err != nil || format != "jpeg" || cfg.Width != 320 || cfg.Height != 160 {
		t.Fatalf("unexpected thumb %v %s %v", cfg, format, err)
	}
}

func TestProcessDoesNotUpscale(t *testing.T) {
	out, err := Process(encodePNG(t, testImage(200, 100)))
	if err != nil || out.Width != 200 || out.Height != 100 {
		t.Fatalf("got %+v %v", out, err)
	}
}

func TestProcessRejectsNonImagesAndSpoofedContent(t *testing.T) {
	cases := map[string][]byte{
		"text":       []byte("hello world, definitely not an image"),
		"html":       []byte("<html><script>alert(1)</script></html>"),
		"gif":        []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"empty":      {},
		"truncated":  encodePNG(t, testImage(100, 100))[:40],
		"jpeg-magic": append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0}, 64)...),
	}
	for name, data := range cases {
		if _, err := Process(data); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestProcessRejectsTinyAndHugeDimensions(t *testing.T) {
	if _, err := Process(encodePNG(t, testImage(32, 32))); err == nil {
		t.Error("tiny image should be rejected")
	}
	// A 9000x9000 grayscale PNG compresses to a few KB: the header check must stop it.
	huge := image.NewGray(image.Rect(0, 0, 9000, 9000))
	if _, err := Process(encodePNG(t, huge)); err == nil {
		t.Error("oversized dimensions should be rejected")
	}
}

func TestProcessFlattensTransparencyOverWhite(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 100, 100)) // fully transparent
	out, err := Process(encodePNG(t, img))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(out.Full))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := decoded.At(50, 50).RGBA()
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Fatalf("expected white background, got %d %d %d", r>>8, g>>8, b>>8)
	}
}

func exifJPEG(t *testing.T, img image.Image, orientation uint16) []byte {
	t.Helper()
	base := encodeJPEG(t, img)
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
	return append(append([]byte{0xFF, 0xD8}, seg...), base[2:]...)
}

func TestProcessAppliesExifOrientation(t *testing.T) {
	data := exifJPEG(t, testImage(200, 100), 6) // rotate 90° CW => 100x200
	if jpegOrientation(data) != 6 {
		t.Fatalf("orientation not parsed: %d", jpegOrientation(data))
	}
	out, err := Process(data)
	if err != nil || out.Width != 100 || out.Height != 200 {
		t.Fatalf("got %+v %v", out, err)
	}
}

func TestOutputCarriesNoExif(t *testing.T) {
	out, err := Process(exifJPEG(t, testImage(200, 100), 1))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Full, []byte("Exif")) {
		t.Fatal("EXIF metadata must be stripped")
	}
}

func TestApplyOrientationMapsPixels(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	src.Set(1, 0, color.RGBA{B: 255, A: 255})
	flipped := applyOrientation(src, 2)
	if r, _, _, _ := flipped.At(1, 0).RGBA(); r == 0 {
		t.Fatal("horizontal flip should move the red pixel to x=1")
	}
	rotated := applyOrientation(src, 6)
	if rotated.Bounds().Dx() != 1 || rotated.Bounds().Dy() != 2 {
		t.Fatalf("unexpected rotated bounds %v", rotated.Bounds())
	}
}
