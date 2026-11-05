package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func solid(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.NRGBA{200, 30, 30, 128}) // semi-transparent
		}
	}
	return img
}

func encode(t *testing.T, img image.Image, f string) []byte {
	t.Helper()
	var b bytes.Buffer
	var err error
	switch f {
	case "png":
		err = png.Encode(&b, img)
	case "jpeg":
		err = jpeg.Encode(&b, img, nil)
	case "gif":
		err = gif.Encode(&b, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestSanitizeAcceptsAndReencodes(t *testing.T) {
	for _, f := range []string{"png", "jpeg"} {
		out, w, h, err := Sanitize(encode(t, solid(3000, 1000), f))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if w != 1280 || h != 426 {
			t.Errorf("%s: expected 1280x426, got %dx%d", f, w, h)
		}
		if cfg, format, err := image.DecodeConfig(bytes.NewReader(out)); err != nil || format != "jpeg" || cfg.Width != w {
			t.Errorf("%s: output must be a JPEG of the reported size (%v %s)", f, err, format)
		}
	}
}

func TestSanitizeKeepsSmallImages(t *testing.T) {
	_, w, h, err := Sanitize(encode(t, solid(64, 48), "png"))
	if err != nil || w != 64 || h != 48 {
		t.Fatalf("got %dx%d err=%v", w, h, err)
	}
}

func TestSanitizeRejects(t *testing.T) {
	png1 := encode(t, solid(10, 10), "png")
	cases := map[string][]byte{
		"empty":     nil,
		"text":      []byte("hello world"),
		"php":       []byte("<?php system($_GET['c']); ?>"),
		"gif":       encode(t, solid(10, 10), "gif"),
		"truncated": png1[:len(png1)/2],
		"huge":      make([]byte, MaxUploadSize+1),
	}
	for name, data := range cases {
		if _, _, _, err := Sanitize(data); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}

func TestSanitizeStripsTrailingPayload(t *testing.T) {
	data := append(encode(t, solid(20, 20), "jpeg"), []byte("<script>alert(1)</script>")...)
	out, _, _, err := Sanitize(data)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("<script>")) {
		t.Fatal("trailing payload survived re-encoding")
	}
}
