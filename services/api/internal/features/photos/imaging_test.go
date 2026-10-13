package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func makePNG(w, h int, alpha uint8) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{200, 30, 30, alpha})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func TestProcessResizesAndReencodesAsJPEG(t *testing.T) {
	out, err := Process(bytes.NewReader(makePNG(3000, 2000, 255)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 1280 || out.Height != 853 {
		t.Fatalf("unexpected dimensions %dx%d", out.Width, out.Height)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out.Data))
	if err != nil || cfg.Width != 1280 {
		t.Fatalf("full image must be a JPEG: %v", err)
	}
	tcfg, err := jpeg.DecodeConfig(bytes.NewReader(out.Thumbnail))
	if err != nil || tcfg.Width != ThumbMaxSide {
		t.Fatalf("thumbnail must be a %dpx JPEG: %v %v", ThumbMaxSide, tcfg, err)
	}
}

func TestProcessKeepsSmallImagesAndFlattensTransparency(t *testing.T) {
	out, err := Process(bytes.NewReader(makePNG(400, 300, 0)))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != 400 || out.Height != 300 {
		t.Fatalf("small images keep their size, got %dx%d", out.Width, out.Height)
	}
	img, _ := jpeg.Decode(bytes.NewReader(out.Data))
	r, g, b, _ := img.At(10, 10).RGBA()
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Fatalf("transparent pixels must become white, got %d %d %d", r>>8, g>>8, b>>8)
	}
}

func TestProcessRejectsUnsafeInput(t *testing.T) {
	var g bytes.Buffer
	_ = gif.Encode(&g, image.NewPaletted(image.Rect(0, 0, 300, 300), color.Palette{color.Black, color.White}), nil)

	cases := map[string]struct {
		data []byte
		want error
	}{
		"empty":          {nil, ErrCorrupt},
		"text":           {[]byte("hello world, definitely not an image"), ErrUnsupportedType},
		"script":         {[]byte("<svg onload=alert(1)></svg>"), ErrUnsupportedType},
		"gif":            {g.Bytes(), ErrUnsupportedType},
		"too small":      {makePNG(100, 100, 255), ErrTooSmall},
		"truncated":      {makePNG(400, 400, 255)[:200], ErrCorrupt},
		"too many bytes": {append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{1}, MaxUploadBytes+10)...), ErrTooLarge},
	}
	for name, c := range cases {
		if _, err := Process(bytes.NewReader(c.data)); err != c.want {
			t.Errorf("%s: want %v, got %v", name, c.want, err)
		}
	}

	// A header claiming huge dimensions is refused before any pixel is decoded.
	huge := makePNG(300, 300, 255)
	copy(huge[16:20], []byte{0, 0, 0x4e, 0x20}) // width 20000
	if _, err := Process(bytes.NewReader(huge)); err == nil {
		t.Error("oversized dimensions must be rejected")
	}
	if _, err := Process(strings.NewReader("")); err == nil {
		t.Error("empty reader must fail")
	}
}
