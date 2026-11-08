package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int, a uint8) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{200, 80, 40, a})
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestProcessImageReencodesAndDownscales(t *testing.T) {
	out, w, h, err := processImage(encodePNG(t, solid(3000, 2000, 255)))
	if err != nil {
		t.Fatal(err)
	}
	if w != 1440 || h != 960 {
		t.Errorf("expected 1440x960, got %dx%d", w, h)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("output must be a decodable JPEG: %v", err)
	}
}

func TestProcessImageStripsTrailingPayload(t *testing.T) {
	var jb bytes.Buffer
	_ = jpeg.Encode(&jb, solid(300, 300, 255), nil)
	polluted := append(jb.Bytes(), []byte("<script>alert(1)</script>")...)
	out, _, _, err := processImage(polluted)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("<script>")) {
		t.Error("re-encoding must drop appended data")
	}
}

func TestProcessImageRejectsBadInput(t *testing.T) {
	cases := map[string][]byte{
		"empty":     nil,
		"text":      []byte("hello world, definitely not an image"),
		"html":      []byte("<html><script>alert(1)</script></html>"),
		"gif":       []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"truncated": encodePNG(t, solid(300, 300, 255))[:60],
		"too small": encodePNG(t, solid(100, 100, 255)),
	}
	for name, data := range cases {
		if _, _, _, err := processImage(data); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestProcessImageFlattensTransparency(t *testing.T) {
	out, _, _, err := processImage(encodePNG(t, solid(300, 300, 0)))
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(10, 10).RGBA()
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Errorf("fully transparent pixels should become white, got %d,%d,%d", r>>8, g>>8, b>>8)
	}
}
