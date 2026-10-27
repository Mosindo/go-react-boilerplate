package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func makePNG(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestNormalizeResizesAndReencodes(t *testing.T) {
	res, err := Normalize(makePNG(2600, 1300))
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != MaxOutputSide || res.Height != MaxOutputSide/2 {
		t.Fatalf("unexpected size %dx%d", res.Width, res.Height)
	}
	if _, format, err := image.DecodeConfig(bytes.NewReader(res.Data)); err != nil || format != "jpeg" {
		t.Fatalf("output must be jpeg, got %q (%v)", format, err)
	}
}

func TestNormalizeKeepsSmallImages(t *testing.T) {
	res, err := Normalize(makePNG(400, 300))
	if err != nil || res.Width != 400 || res.Height != 300 {
		t.Fatalf("got %+v, %v", res, err)
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 300, 300)), nil)
	if _, err := Normalize(buf.Bytes()); err != nil {
		t.Fatalf("jpeg input: %v", err)
	}
}

func TestNormalizeRejectsBadInput(t *testing.T) {
	var g bytes.Buffer
	_ = gif.Encode(&g, image.NewPaletted(image.Rect(0, 0, 300, 300), color.Palette{color.Black, color.White}), nil)

	cases := map[string]struct {
		data []byte
		want error
	}{
		"empty":          {nil, ErrCorrupt},
		"text":           {[]byte("<?php echo 1; ?>"), ErrUnsupported},
		"gif":            {g.Bytes(), ErrUnsupported},
		"tiny":           {makePNG(50, 50), ErrTooSmall},
		"truncated png":  {makePNG(400, 400)[:200], ErrCorrupt},
		"too big upload": {make([]byte, MaxUploadBytes+1), ErrCorrupt},
	}
	for name, tc := range cases {
		if _, err := Normalize(tc.data); !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", name, err, tc.want)
		}
	}
}
