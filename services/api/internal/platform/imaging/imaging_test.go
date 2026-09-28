package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func jpegBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func pngBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func size(t *testing.T, data []byte) (int, int, string) {
	t.Helper()
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not a decodable image: %v", err)
	}
	return cfg.Width, cfg.Height, format
}

func TestProcessAcceptsJPEGAndPNG(t *testing.T) {
	src := solid(300, 200, color.RGBA{200, 30, 30, 255})
	for name, data := range map[string][]byte{"jpeg": jpegBytes(t, src), "png": pngBytes(t, src)} {
		res, err := Process(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		w, h, format := size(t, res.JPEG)
		if format != "jpeg" || w != 300 || h != 200 || res.Width != 300 || res.Height != 200 {
			t.Errorf("%s: got %dx%d %s (%dx%d)", name, w, h, format, res.Width, res.Height)
		}
	}
}

func TestProcessRejectsNonImages(t *testing.T) {
	var gifBuf bytes.Buffer
	_ = gif.Encode(&gifBuf, image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{color.Black, color.White}), nil)
	cases := map[string][]byte{
		"empty":            nil,
		"text":             []byte("hello world, definitely not an image"),
		"html renamed jpg": []byte("<html><script>alert(1)</script></html>"),
		"gif renamed jpg":  gifBuf.Bytes(),
		"pdf":              []byte("%PDF-1.4 fake"),
		"svg":              []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
		"zip":              []byte("PK\x03\x04rest"),
		"jpeg magic only":  append([]byte("\xff\xd8\xff\xe0"), bytes.Repeat([]byte{0}, 50)...),
		"truncated jpeg":   jpegBytes(t, solid(64, 64, color.RGBA{1, 2, 3, 255}))[:120],
		"truncated png":    pngBytes(t, solid(64, 64, color.RGBA{1, 2, 3, 255}))[:60],
		"png magic only":   []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("x", 30)),
		"php polyglot":     append([]byte("<?php system($_GET['c']); ?>"), jpegBytes(t, solid(8, 8, color.RGBA{}))...),
	}
	for name, data := range cases {
		if _, err := Process(data); err == nil {
			t.Errorf("%s must be rejected", name)
		} else if !errors.Is(err, ErrUnsupported) && !errors.Is(err, ErrCorrupt) {
			t.Errorf("%s: unexpected error %v", name, err)
		}
	}
}

func TestProcessRejectsDecompressionBombs(t *testing.T) {
	// 5000x5000 = 25M px > MaxSourcePixels; a flat PNG compresses to a few KB.
	big := image.NewGray(image.Rect(0, 0, 5000, 5000))
	if _, err := Process(pngBytes(t, big)); !errors.Is(err, ErrTooLarge) {
		t.Errorf("expected ErrTooLarge, got %v", err)
	}
}

func TestDownscaleKeepsAspectRatioAndLimit(t *testing.T) {
	cases := []struct{ w, h, wantW, wantH int }{
		{2560, 1280, 1280, 640},
		{1000, 3000, 426, 1280},
		{1281, 1281, 1280, 1280},
		{1280, 100, 1280, 100},
		{50, 40, 50, 40},
		{4000, 1, 1280, 1},
	}
	for _, c := range cases {
		out := Downscale(solid(c.w, c.h, color.RGBA{10, 20, 30, 255}), MaxSide)
		if out.Bounds().Dx() != c.wantW || out.Bounds().Dy() != c.wantH {
			t.Errorf("%dx%d -> %dx%d, want %dx%d", c.w, c.h, out.Bounds().Dx(), out.Bounds().Dy(), c.wantW, c.wantH)
		}
	}
}

func TestDownscaleAveragesPixels(t *testing.T) {
	// 2x2 checkerboard of black/white collapsed to 1x1 must be mid grey.
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.SetRGBA(0, 0, color.RGBA{0, 0, 0, 255})
	src.SetRGBA(1, 0, color.RGBA{255, 255, 255, 255})
	src.SetRGBA(0, 1, color.RGBA{255, 255, 255, 255})
	src.SetRGBA(1, 1, color.RGBA{0, 0, 0, 255})
	out := Downscale(src, 1)
	if out.Bounds().Dx() != 1 {
		t.Fatalf("size %v", out.Bounds())
	}
	c := out.RGBAAt(0, 0)
	if c.R < 126 || c.R > 128 || c.G != c.R || c.B != c.R || c.A != 255 {
		t.Errorf("expected mid grey, got %+v", c)
	}
}

func TestTransparencyIsFlattenedOverWhite(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 8, 8)) // fully transparent
	res, err := Process(pngBytes(t, src))
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(res.JPEG))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(4, 4).RGBA()
	if r>>8 < 245 || g>>8 < 245 || b>>8 < 245 {
		t.Errorf("transparent pixels must become white, got %d %d %d", r>>8, g>>8, b>>8)
	}
}

// exifJPEG injects an APP1 Exif segment with the given orientation and marker text.
func exifJPEG(t *testing.T, base []byte, orientation int, littleEndian bool, marker string) []byte {
	t.Helper()
	var tiff []byte
	if littleEndian {
		tiff = []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, byte(orientation), 0, 0, 0, 0, 0, 0, 0}
	} else {
		tiff = []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, byte(orientation), 0, 0, 0, 0, 0, 0}
	}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	payload = append(payload, []byte(marker)...)
	seg := []byte{0xFF, 0xE1, byte((len(payload) + 2) >> 8), byte((len(payload) + 2) & 0xFF)}
	seg = append(seg, payload...)
	out := append([]byte{}, base[:2]...)
	out = append(out, seg...)
	return append(out, base[2:]...)
}

func TestJPEGOrientationParsing(t *testing.T) {
	base := jpegBytes(t, solid(10, 6, color.RGBA{9, 9, 9, 255}))
	for _, le := range []bool{true, false} {
		for o := 1; o <= 8; o++ {
			if got := JPEGOrientation(exifJPEG(t, base, o, le, "")); got != o {
				t.Errorf("orientation %d (LE=%v) parsed as %d", o, le, got)
			}
		}
	}
	if got := JPEGOrientation(base); got != 1 {
		t.Errorf("no exif => 1, got %d", got)
	}
	if got := JPEGOrientation([]byte("junk")); got != 1 {
		t.Errorf("junk => 1, got %d", got)
	}
	bad := exifJPEG(t, base, 9, true, "")
	if got := JPEGOrientation(bad); got != 1 {
		t.Errorf("out of range orientation => 1, got %d", got)
	}
}

func TestProcessAppliesOrientationAndStripsMetadata(t *testing.T) {
	// 40x20 image, red left half / blue right half
	src := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			if x < 20 {
				src.SetRGBA(x, y, color.RGBA{255, 0, 0, 255})
			} else {
				src.SetRGBA(x, y, color.RGBA{0, 0, 255, 255})
			}
		}
	}
	base := jpegBytes(t, src)
	withExif := exifJPEG(t, base, 6, false, "GPSLatitude=48.85;Make=SecretCam") // 6 = rotate 90 CW
	if !bytes.Contains(withExif, []byte("GPSLatitude")) {
		t.Fatal("setup")
	}
	res, err := Process(withExif)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(res.JPEG, []byte("Exif")) || bytes.Contains(res.JPEG, []byte("GPS")) || bytes.Contains(res.JPEG, []byte("SecretCam")) {
		t.Error("metadata must be gone")
	}
	if JPEGOrientation(res.JPEG) != 1 {
		t.Error("output must carry no orientation tag")
	}
	if res.Width != 20 || res.Height != 40 {
		t.Fatalf("rotated size should be 20x40, got %dx%d", res.Width, res.Height)
	}
	img, _ := jpeg.Decode(bytes.NewReader(res.JPEG))
	// after 90 CW rotation the left (red) half ends up on top
	tr, _, tb, _ := img.At(10, 5).RGBA()
	br, _, bb, _ := img.At(10, 35).RGBA()
	if !(tr>>8 > 200 && tb>>8 < 60) || !(br>>8 < 60 && bb>>8 > 200) {
		t.Errorf("unexpected rotation: top=(%d,%d) bottom=(%d,%d)", tr>>8, tb>>8, br>>8, bb>>8)
	}
}

func TestApplyOrientationDimensions(t *testing.T) {
	src := solid(6, 4, color.RGBA{1, 1, 1, 255})
	for o := 1; o <= 8; o++ {
		out := applyOrientation(src, o)
		w, h := out.Bounds().Dx(), out.Bounds().Dy()
		if o >= 5 && (w != 4 || h != 6) || o < 5 && (w != 6 || h != 4) {
			t.Errorf("orientation %d: %dx%d", o, w, h)
		}
	}
}
