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
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	return img
}

func TestNormalizeImageResizesAndReencodesAsJPEG(t *testing.T) {
	var src bytes.Buffer
	if err := png.Encode(&src, solid(3200, 1600)); err != nil {
		t.Fatal(err)
	}
	out, w, h, err := normalizeImage(src.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if w != 1600 || h != 800 {
		t.Fatalf("got %dx%d", w, h)
	}
	if _, format, err := image.DecodeConfig(bytes.NewReader(out)); err != nil || format != "jpeg" {
		t.Fatalf("output must be a JPEG, got %q (%v)", format, err)
	}
}

func TestNormalizeImageFlattensTransparency(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 300, 300)) // fully transparent
	var src bytes.Buffer
	_ = png.Encode(&src, img)
	out, _, _, err := normalizeImage(src.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	decoded, _ := jpeg.Decode(bytes.NewReader(out))
	r, g, b, _ := decoded.At(10, 10).RGBA()
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Fatalf("transparent pixels should become white, got %v", color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 255})
	}
}

func TestNormalizeImageRejectsBadInput(t *testing.T) {
	var small, truncated bytes.Buffer
	_ = png.Encode(&small, solid(100, 100))
	_ = png.Encode(&truncated, solid(400, 400))
	cases := map[string][]byte{
		"text":      []byte("hello world"),
		"html":      []byte("<script>alert(1)</script>"),
		"too small": small.Bytes(),
		"truncated": truncated.Bytes()[:truncated.Len()/2],
		"gif":       []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
	}
	for name, data := range cases {
		if _, _, _, err := normalizeImage(data); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}
