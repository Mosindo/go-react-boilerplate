package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestProcessImageResizesAndReencodes(t *testing.T) {
	out, err := ProcessImage(encodePNG(t, solid(2000, 1000)))
	if err != nil {
		t.Fatal(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil || format != "jpeg" || cfg.Width != maxOutputEdge || cfg.Height != 640 {
		t.Fatalf("unexpected output %v %s %+v", err, format, cfg)
	}
}

func TestProcessImageRejects(t *testing.T) {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, solid(100, 100), nil)
	cases := map[string][]byte{
		"empty":         nil,
		"text":          []byte("<script>alert(1)</script>"),
		"too small":     buf.Bytes(),
		"gif header":    []byte("GIF89a\x01\x00\x01\x00"),
		"png truncated": encodePNG(t, solid(300, 300))[:200],
		"oversize":      make([]byte, maxUploadBytes+1),
	}
	for name, data := range cases {
		if _, err := ProcessImage(data); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}

func TestProcessImageFlattensAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 300, 300)) // fully transparent
	out, err := ProcessImage(encodePNG(t, img))
	if err != nil {
		t.Fatal(err)
	}
	dec, _ := jpeg.Decode(bytes.NewReader(out))
	r, g, b, _ := dec.At(150, 150).RGBA()
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Fatalf("transparent pixel should become white, got %d %d %d", r>>8, g>>8, b>>8)
	}
}

func TestOrientationRotates(t *testing.T) {
	src := solid(40, 20)
	if got := applyOrientation(src, 6).Bounds(); got.Dx() != 20 || got.Dy() != 40 {
		t.Fatalf("orientation 6 must swap dimensions, got %v", got)
	}
	if got := applyOrientation(src, 3).Bounds(); got.Dx() != 40 {
		t.Fatalf("orientation 3 keeps dimensions, got %v", got)
	}
	if jpegOrientation([]byte{0xFF, 0xD8, 0xFF, 0xD9}) != 1 || jpegOrientation(nil) != 1 {
		t.Fatal("missing EXIF must default to 1")
	}
}
