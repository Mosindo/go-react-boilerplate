package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func encode(t *testing.T, w, h int, asPNG bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{uint8(x), uint8(y), 50, 128})
		}
	}
	var b bytes.Buffer
	if asPNG {
		_ = png.Encode(&b, img)
	} else {
		_ = jpeg.Encode(&b, img, nil)
	}
	return b.Bytes()
}

func TestProcessDownscalesAndReencodes(t *testing.T) {
	out, err := Process(bytes.NewReader(encode(t, 3000, 2000, true)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1080 || out.Height != 720 {
		t.Errorf("size = %dx%d", out.Width, out.Height)
	}
	if len(out.Data) < 2 || out.Data[0] != 0xFF || out.Data[1] != 0xD8 {
		t.Error("output must be a JPEG")
	}
	if _, _, err := image.Decode(bytes.NewReader(out.Data)); err != nil {
		t.Errorf("output must decode: %v", err)
	}
}

func TestProcessKeepsSmallImages(t *testing.T) {
	out, err := Process(bytes.NewReader(encode(t, 400, 300, false)))
	if err != nil || out.Width != 400 || out.Height != 300 {
		t.Fatalf("%v %dx%d", err, out.Width, out.Height)
	}
}

func TestProcessRejects(t *testing.T) {
	cases := map[string][]byte{
		"text":       []byte("hello world"),
		"php":        []byte("<?php system($_GET['c']); ?>"),
		"truncated":  encode(t, 400, 400, false)[:100],
		"tiny":       encode(t, 100, 100, false),
		"svg":        []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"empty":      {},
		"jpeg magic": append([]byte{0xFF, 0xD8, 0xFF}, bytes.Repeat([]byte("A"), 500)...),
	}
	for name, data := range cases {
		if _, err := Process(bytes.NewReader(data)); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	if _, err := Process(strings.NewReader(strings.Repeat("a", MaxUploadBytes+10))); err == nil {
		t.Error("oversized upload must be rejected")
	}
}
