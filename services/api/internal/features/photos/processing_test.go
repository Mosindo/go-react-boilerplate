package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func makeJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 80, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessImageDownscalesAndReencodes(t *testing.T) {
	out, err := processImage(makeJPEG(t, 2400, 1200))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1080 || out.Height != 540 {
		t.Fatalf("expected 1080x540, got %dx%d", out.Width, out.Height)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(out.Data))
	if err != nil || format != "jpeg" || cfg.Width != 1080 {
		t.Fatalf("output must be a 1080px JPEG, got %v %v %v", cfg, format, err)
	}
}

func TestProcessImageDropsTrailingPayloadAndMetadata(t *testing.T) {
	raw := append(makeJPEG(t, 400, 400), []byte("<?php system($_GET['c']); ?>")...)
	out, err := processImage(raw)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Data, []byte("<?php")) {
		t.Fatal("re-encoding must discard anything appended to the image")
	}
}

func TestProcessImageFlattensTransparentPNG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 300, 300)) // fully transparent
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	out, err := processImage(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := jpeg.Decode(bytes.NewReader(out.Data))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := decoded.At(150, 150).RGBA()
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Fatal("transparent pixels must become white, not black")
	}
}

func TestProcessImageRejects(t *testing.T) {
	cases := map[string][]byte{
		"text":        []byte("hello world, definitely not an image"),
		"empty":       {},
		"gif":         append([]byte("GIF89a"), make([]byte, 64)...),
		"svg":         []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"truncated":   makeJPEG(t, 400, 400)[:200],
		"too small":   makeJPEG(t, 150, 150),
		"html in png": append([]byte("\x89PNG\r\n\x1a\n"), []byte("<html>")...),
	}
	for name, raw := range cases {
		if _, err := processImage(raw); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}

func TestProcessImageRejectsDecompressionBomb(t *testing.T) {
	// A valid PNG header claiming 30000x30000 pixels: must fail before any decode.
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 30000, 1))); err != nil {
		t.Fatal(err)
	}
	if _, err := processImage(buf.Bytes()); err == nil {
		t.Fatal("oversized dimensions must be rejected")
	}
}
