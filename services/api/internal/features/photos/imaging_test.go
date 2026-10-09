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
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestProcessDownscalesAndReencodes(t *testing.T) {
	out, err := Process(encodeJPEG(t, solid(3000, 2000, color.RGBA{200, 10, 10, 255})))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1440 || out.Height != 960 {
		t.Fatalf("unexpected size %dx%d", out.Width, out.Height)
	}
	if _, format, err := image.DecodeConfig(bytes.NewReader(out.Data)); err != nil || format != "jpeg" {
		t.Fatalf("output must be a JPEG, got %v %v", format, err)
	}
}

func TestProcessConvertsPNGAndFlattensTransparency(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 300, 300)) // fully transparent
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	out, err := Process(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, _ := image.Decode(bytes.NewReader(out.Data))
	r, g, bl, _ := decoded.At(150, 150).RGBA()
	if r>>8 < 250 || g>>8 < 250 || bl>>8 < 250 {
		t.Fatalf("transparent areas must become white, got %d %d %d", r>>8, g>>8, bl>>8)
	}
}

func TestProcessStripsTrailingPayloadAndMetadata(t *testing.T) {
	raw := append(encodeJPEG(t, solid(300, 300, color.White)), []byte("<?php evil() ?>")...)
	out, err := Process(raw)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Data, []byte("evil")) {
		t.Fatal("trailing payload survived re-encoding")
	}
}

func TestProcessRejectsInvalidInput(t *testing.T) {
	cases := map[string][]byte{
		"text":      []byte("hello world"),
		"php":       []byte("<?php echo 1; ?>"),
		"empty":     {},
		"truncated": encodeJPEG(t, solid(300, 300, color.White))[:100],
		"gif":       append([]byte("GIF89a"), make([]byte, 64)...),
	}
	for name, raw := range cases {
		if _, err := Process(raw); err == nil {
			t.Errorf("%s: should be rejected", name)
		}
	}
	if _, err := Process(encodeJPEG(t, solid(100, 100, color.White))); err != ErrImageTooSmall {
		t.Errorf("small image: got %v", err)
	}
}

func TestProcessRejectsDecompressionBombs(t *testing.T) {
	// A PNG header claiming 30000x30000 pixels must be refused from its header alone.
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewGray(image.Rect(0, 0, 1, 1)))
	raw := b.Bytes()
	binary.BigEndian.PutUint32(raw[16:20], 30000)
	binary.BigEndian.PutUint32(raw[20:24], 30000)
	if _, err := Process(raw); err == nil {
		t.Fatal("oversized dimensions must be rejected")
	}
}

func TestOrientationIsApplied(t *testing.T) {
	// 300x200 image, left half red / right half blue, rotated 90deg CW by EXIF (6): result is 200x300, red on top.
	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 300; x++ {
			if x < 150 {
				img.Set(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				img.Set(x, y, color.RGBA{0, 0, 255, 255})
			}
		}
	}
	raw := withOrientation(encodeJPEG(t, img), 6)
	if got := jpegOrientation(raw); got != 6 {
		t.Fatalf("orientation parse: got %d", got)
	}
	out, err := Process(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 200 || out.Height != 300 {
		t.Fatalf("expected rotated dimensions, got %dx%d", out.Width, out.Height)
	}
	decoded, _, _ := image.Decode(bytes.NewReader(out.Data))
	top, _, _, _ := decoded.At(100, 20).RGBA()
	bottom, _, _, _ := decoded.At(100, 280).RGBA()
	if top>>8 < 200 || bottom>>8 > 60 {
		t.Fatalf("red half should be on top after rotation (top r=%d bottom r=%d)", top>>8, bottom>>8)
	}
}

func TestJPEGOrientationHandlesGarbage(t *testing.T) {
	for _, raw := range [][]byte{nil, {0xFF}, {0xFF, 0xD8}, {0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x04, 'E', 'x'}, []byte("not a jpeg")} {
		if got := jpegOrientation(raw); got != 1 {
			t.Errorf("garbage %v gave %d", raw, got)
		}
	}
}

// withOrientation injects a minimal EXIF APP1 segment right after SOI.
func withOrientation(jpegData []byte, orientation uint16) []byte {
	var tiff bytes.Buffer
	tiff.WriteString("MM")
	_ = binary.Write(&tiff, binary.BigEndian, uint16(42))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(8))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(1)) // one IFD entry
	_ = binary.Write(&tiff, binary.BigEndian, uint16(0x0112))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(3))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(1))
	_ = binary.Write(&tiff, binary.BigEndian, orientation)
	_ = binary.Write(&tiff, binary.BigEndian, uint16(0))
	_ = binary.Write(&tiff, binary.BigEndian, uint32(0))

	seg := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	var out bytes.Buffer
	out.Write([]byte{0xFF, 0xD8, 0xFF, 0xE1})
	_ = binary.Write(&out, binary.BigEndian, uint16(len(seg)+2))
	out.Write(seg)
	out.Write(jpegData[2:])
	return out.Bytes()
}
