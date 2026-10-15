package files

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

// withOrientation injects an EXIF APP1 segment carrying the orientation tag.
func withOrientation(t *testing.T, jpegBytes []byte, orientation uint16) []byte {
	t.Helper()
	tiff := new(bytes.Buffer)
	tiff.WriteString("MM")
	_ = binary.Write(tiff, binary.BigEndian, uint16(42))
	_ = binary.Write(tiff, binary.BigEndian, uint32(8))
	_ = binary.Write(tiff, binary.BigEndian, uint16(1)) // one IFD entry
	_ = binary.Write(tiff, binary.BigEndian, uint16(0x0112))
	_ = binary.Write(tiff, binary.BigEndian, uint16(3))
	_ = binary.Write(tiff, binary.BigEndian, uint32(1))
	_ = binary.Write(tiff, binary.BigEndian, orientation)
	_ = binary.Write(tiff, binary.BigEndian, uint16(0))
	_ = binary.Write(tiff, binary.BigEndian, uint32(0))

	seg := new(bytes.Buffer)
	seg.WriteString("Exif\x00\x00")
	seg.Write(tiff.Bytes())

	out := new(bytes.Buffer)
	out.Write(jpegBytes[:2])
	out.Write([]byte{0xFF, 0xE1})
	_ = binary.Write(out, binary.BigEndian, uint16(seg.Len()+2))
	out.Write(seg.Bytes())
	out.Write(jpegBytes[2:])
	return out.Bytes()
}

func TestProcessImageRejectsNonImages(t *testing.T) {
	for name, data := range map[string][]byte{
		"text":       []byte("hello world, definitely not an image"),
		"php":        []byte("<?php echo 1; ?>"),
		"empty":      {},
		"svg":        []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"gif header": []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"truncated":  encodePNG(t, solid(400, 400, color.White))[:100],
	} {
		if _, err := processImage(data); !errors.Is(err, errUnsupportedImage) {
			t.Errorf("%s: expected errUnsupportedImage, got %v", name, err)
		}
	}
}

func TestProcessImageSizeGuards(t *testing.T) {
	if _, err := processImage(encodePNG(t, solid(100, 100, color.White))); !errors.Is(err, errImageTooSmall) {
		t.Errorf("tiny image: %v", err)
	}
	// A 13000x13000 PNG compresses to a few KB but would need ~675 MB to decode.
	bomb := encodePNG(t, image.NewGray(image.Rect(0, 0, 13000, 13000)))
	if len(bomb) > 10<<20 {
		t.Skipf("fixture unexpectedly large: %d bytes", len(bomb))
	}
	if _, err := processImage(bomb); !errors.Is(err, errImageTooLarge) {
		t.Errorf("decompression bomb must be refused before decoding, got %v", err)
	}
}

func TestProcessImageResizesAndStripsMetadata(t *testing.T) {
	src := withOrientation(t, encodeJPEG(t, solid(3000, 2000, color.RGBA{200, 50, 50, 255})), 1)
	got, err := processImage(src)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != fullSize || got.Height != 720 {
		t.Errorf("expected 1080x720, got %dx%d", got.Width, got.Height)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(got.Full))
	if err != nil || format != "jpeg" || cfg.Width != 1080 {
		t.Errorf("full: %v %s %+v", err, format, cfg)
	}
	thumb, _, err := image.DecodeConfig(bytes.NewReader(got.Thumb))
	if err != nil || thumb.Width != thumbSize {
		t.Errorf("thumb: %v %+v", err, thumb)
	}
	if bytes.Contains(got.Full, []byte("Exif")) {
		t.Error("EXIF must not survive re-encoding")
	}
	if len(got.Full) >= len(src) {
		t.Errorf("output (%d) should be lighter than the 3000px source (%d)", len(got.Full), len(src))
	}
}

func TestProcessImageAppliesExifOrientation(t *testing.T) {
	landscape := encodeJPEG(t, solid(600, 400, color.RGBA{10, 120, 200, 255}))
	for orientation, want := range map[uint16][2]int{1: {600, 400}, 3: {600, 400}, 6: {400, 600}, 8: {400, 600}} {
		got, err := processImage(withOrientation(t, landscape, orientation))
		if err != nil {
			t.Fatalf("orientation %d: %v", orientation, err)
		}
		if got.Width != want[0] || got.Height != want[1] {
			t.Errorf("orientation %d: expected %v, got %dx%d", orientation, want, got.Width, got.Height)
		}
	}
}

func TestOrientationPixelsMoveToTheRightCorner(t *testing.T) {
	// 4x2 image with a red pixel in the top-left corner.
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	rotated := applyOrientation(img, 6) // 90 degrees clockwise: top-left goes to top-right
	if rotated.Bounds().Dx() != 2 || rotated.Bounds().Dy() != 4 {
		t.Fatalf("unexpected size %v", rotated.Bounds())
	}
	if r, _, _, _ := rotated.At(1, 0).RGBA(); r>>8 != 255 {
		t.Error("the red pixel should land in the top-right corner after a clockwise rotation")
	}
}

func TestTransparencyIsFlattenedOnWhite(t *testing.T) {
	transparent := image.NewNRGBA(image.Rect(0, 0, 300, 300))
	got, err := processImage(encodePNG(t, transparent))
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := image.Decode(bytes.NewReader(got.Full))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := decoded.At(150, 150).RGBA()
	if r>>8 < 250 || g>>8 < 250 || b>>8 < 250 {
		t.Errorf("transparent pixels should become white, got %d %d %d", r>>8, g>>8, b>>8)
	}
}
