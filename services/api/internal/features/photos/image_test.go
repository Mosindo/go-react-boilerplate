package photos

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"testing"
)

func solid(w, h int, c color.Color) image.Image {
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

// withOrientation injects a minimal EXIF APP1 segment (plus a GPS-looking payload) after SOI.
func withOrientation(data []byte, orientation uint16) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1}
	tiff = binary.BigEndian.AppendUint16(tiff, orientation)
	tiff = append(tiff, 0, 0, 0, 0, 0, 0)
	seg := append([]byte("Exif\x00\x00"), tiff...)
	app1 := []byte{0xFF, 0xE1}
	app1 = binary.BigEndian.AppendUint16(app1, uint16(len(seg)+2))
	app1 = append(app1, seg...)
	out := append([]byte{}, data[:2]...)
	out = append(out, app1...)
	return append(out, data[2:]...)
}

func TestProcessImageRejectsNonImages(t *testing.T) {
	var g bytes.Buffer
	_ = gif.Encode(&g, solid(300, 300, color.White), nil)
	for name, data := range map[string][]byte{
		"empty": nil, "text": []byte("hello world, definitely not an image"), "php": []byte("<?php system($_GET['c']); ?>"),
		"gif": g.Bytes(), "svg": []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
	} {
		if _, err := processImage(data); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}

func TestProcessImageBounds(t *testing.T) {
	if _, err := processImage(encodeJPEG(t, solid(100, 100, color.White))); err == nil {
		t.Error("too small must be rejected")
	}
	out, err := processImage(encodeJPEG(t, solid(2400, 1200, color.RGBA{R: 10, G: 20, B: 30, A: 255})))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1600 || out.Height != 800 {
		t.Errorf("got %dx%d", out.Width, out.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Errorf("output must be a valid JPEG: %v", err)
	}
}

func TestProcessImageStripsMetadataAndFixesOrientation(t *testing.T) {
	// 400x300 landscape tagged "rotate 90° CW" (6) must come out as 300x400 portrait without EXIF.
	src := withOrientation(encodeJPEG(t, solid(400, 300, color.RGBA{R: 200, A: 255})), 6)
	if jpegOrientation(src) != 6 {
		t.Fatal("test fixture should carry orientation 6")
	}
	out, err := processImage(src)
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 300 || out.Height != 400 {
		t.Errorf("orientation not applied: %dx%d", out.Width, out.Height)
	}
	if bytes.Contains(out.Data, []byte("Exif")) {
		t.Error("EXIF must be stripped")
	}
	if jpegOrientation(out.Data) != 1 {
		t.Error("output should not carry an orientation tag")
	}
}

func TestJPEGOrientationMalformed(t *testing.T) {
	for _, data := range [][]byte{nil, {0xFF}, {0xFF, 0xD8}, {0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x04}, []byte("not a jpeg")} {
		if got := jpegOrientation(data); got != 1 {
			t.Errorf("malformed input should default to 1, got %d", got)
		}
	}
}
