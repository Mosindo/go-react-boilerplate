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

func testJPEG(w, h int, marker color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	// Top-left 20x20 block is the marker used to check orientation.
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, marker)
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	return buf.Bytes()
}

// withOrientation injects an EXIF APP1 segment holding the orientation tag.
func withOrientation(jpg []byte, orientation uint16) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0}
	binary.BigEndian.PutUint16(tiff[18:], orientation)
	seg := append([]byte("Exif\x00\x00"), tiff...)
	size := len(seg) + 2
	out := []byte{0xFF, 0xD8, 0xFF, 0xE1, byte(size >> 8), byte(size)}
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func decode(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not a JPEG: %v", err)
	}
	return img
}

func isRed(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r>>8 > 200 && g>>8 < 80 && b>>8 < 80
}

func TestProcessImageDownscalesAndKeepsAspectRatio(t *testing.T) {
	out, err := processImage(testJPEG(2880, 1920, color.RGBA{255, 0, 0, 255}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1440 || out.Height != 960 {
		t.Fatalf("expected 1440x960, got %dx%d", out.Width, out.Height)
	}
	if decode(t, out.Data).Bounds().Dx() != 1440 {
		t.Fatal("encoded size must match reported size")
	}

	small, err := processImage(testJPEG(300, 400, color.RGBA{255, 0, 0, 255}))
	if err != nil || small.Width != 300 || small.Height != 400 {
		t.Fatalf("small images keep their size: %+v %v", small, err)
	}
}

func TestProcessImageAppliesExifOrientation(t *testing.T) {
	src := testJPEG(400, 300, color.RGBA{255, 0, 0, 255}) // red marker in the top-left corner

	cases := []struct {
		orientation   uint16
		w, h          int
		redX, redY    int // a pixel that must be red after orienting
		notRedX, notY int // a pixel that must be white
	}{
		{1, 400, 300, 5, 5, 395, 295},
		{3, 400, 300, 395, 295, 5, 5}, // rotated 180°
		{6, 300, 400, 295, 5, 5, 395}, // rotate 90° CW: marker goes to the top-right
		{8, 300, 400, 5, 395, 295, 5}, // rotate 90° CCW: marker goes to the bottom-left
		{2, 400, 300, 395, 5, 5, 5},   // mirrored horizontally
	}
	for _, tc := range cases {
		out, err := processImage(withOrientation(src, tc.orientation))
		if err != nil {
			t.Fatalf("orientation %d: %v", tc.orientation, err)
		}
		if out.Width != tc.w || out.Height != tc.h {
			t.Errorf("orientation %d: expected %dx%d, got %dx%d", tc.orientation, tc.w, tc.h, out.Width, out.Height)
			continue
		}
		img := decode(t, out.Data)
		if !isRed(img.At(tc.redX, tc.redY)) {
			t.Errorf("orientation %d: marker not where expected", tc.orientation)
		}
		if isRed(img.At(tc.notRedX, tc.notY)) {
			t.Errorf("orientation %d: unexpected marker position", tc.orientation)
		}
	}
}

func TestProcessImageFlattensTransparency(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 300, 300)) // fully transparent
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	out, err := processImage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := decode(t, out.Data).At(150, 150).RGBA()
	if r>>8 < 250 || g>>8 < 250 || b>>8 < 250 {
		t.Fatalf("transparent pixels must become white, got %d,%d,%d", r>>8, g>>8, b>>8)
	}
}

func TestProcessImageRejects(t *testing.T) {
	cases := map[string][]byte{
		"empty":     nil,
		"text":      []byte("hello world"),
		"html":      []byte("<html><body>x</body></html>"),
		"truncated": testJPEG(400, 400, color.RGBA{})[:300],
		"gif":       []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
	}
	for name, data := range cases {
		if _, err := processImage(data); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if _, err := processImage(testJPEG(100, 100, color.RGBA{})); err != ErrImageTooSmall {
		t.Errorf("tiny image: got %v", err)
	}
}

func TestExifOrientationParsingIsRobust(t *testing.T) {
	for _, data := range [][]byte{nil, {0xFF}, {0xFF, 0xD8}, {0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x04}, bytes.Repeat([]byte{0xFF, 0xD8}, 50)} {
		if got := exifOrientation(data); got != 1 {
			t.Errorf("malformed input must give orientation 1, got %d", got)
		}
	}
	if got := exifOrientation(withOrientation(testJPEG(300, 300, color.RGBA{}), 9)); got != 1 {
		t.Errorf("out-of-range orientation must be ignored, got %d", got)
	}
}
