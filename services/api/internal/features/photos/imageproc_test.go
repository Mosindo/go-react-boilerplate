package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func solid(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 120, A: 255})
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

func TestProcessAcceptsJPEGAndPNG(t *testing.T) {
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, solid(400, 300)); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"jpeg": encodeJPEG(t, solid(400, 300)), "png": pngBuf.Bytes()} {
		out, err := Process(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if out.Width != 400 || out.Height != 300 {
			t.Errorf("%s: unexpected size %dx%d", name, out.Width, out.Height)
		}
		if !bytes.HasPrefix(out.Data, []byte{0xFF, 0xD8, 0xFF}) {
			t.Errorf("%s: output must be a JPEG", name)
		}
	}
}

func TestProcessDownscalesLargeImages(t *testing.T) {
	out, err := Process(bytes.NewReader(encodeJPEG(t, solid(2600, 1300))))
	if err != nil {
		t.Fatal(err)
	}
	if out.Width != outputMaxSide || out.Height != 640 {
		t.Fatalf("expected 1280x640, got %dx%d", out.Width, out.Height)
	}
}

func TestProcessRejectsDangerousInput(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want error
	}{
		"html":        {[]byte("<html><script>alert(1)</script></html>"), ErrUnsupportedType},
		"svg":         {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>1</script></svg>`), ErrUnsupportedType},
		"gif":         {[]byte("GIF89a......"), ErrUnsupportedType},
		"empty":       {nil, ErrUnsupportedType},
		"truncated":   {[]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}, ErrCorruptImage},
		"too small":   {encodeJPEG(t, solid(50, 50)), ErrBadDimensions},
		"oversize":    {append(encodeJPEG(t, solid(300, 300)), make([]byte, MaxUploadBytes)...), ErrTooLarge},
		"php in jpeg": {append([]byte{0xFF, 0xD8, 0xFF}, []byte("<?php system($_GET['c']); ?>")...), ErrCorruptImage},
	}
	for name, tc := range cases {
		if _, err := Process(bytes.NewReader(tc.data)); !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", name, err, tc.want)
		}
	}
}

func TestProcessStripsAppendedPayload(t *testing.T) {
	payload := "<?php evil(); ?>"
	data := append(encodeJPEG(t, solid(300, 300)), []byte(payload)...)
	out, err := Process(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out.Data), "evil") {
		t.Fatal("trailing bytes must not survive re-encoding")
	}
}

func TestOrientationTransforms(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	src.Set(0, 0, color.RGBA{R: 255, A: 255}) // top-left marker
	rotated := applyOrientation(src, 6)       // rotate 90° clockwise
	if rotated.Bounds().Dx() != 2 || rotated.Bounds().Dy() != 4 {
		t.Fatalf("orientation 6 must swap dimensions, got %v", rotated.Bounds())
	}
	if c := rotated.RGBAAt(1, 0); c.R != 255 {
		t.Fatalf("top-left pixel should land top-right after a clockwise turn, got %+v", c)
	}
	if same := applyOrientation(src, 1); same != src {
		t.Fatal("orientation 1 must be a no-op")
	}
}

func TestExifOrientationParsing(t *testing.T) {
	// SOI + APP1(Exif, little endian, one IFD entry: orientation = 6)
	tiff := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, 6, 0, 0, 0, 0, 0, 0, 0}
	app1 := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1, byte((len(app1) + 2) >> 8), byte(len(app1) + 2)}
	data := append([]byte{0xFF, 0xD8}, append(seg, app1...)...)
	data = append(data, 0xFF, 0xDA)
	if got := exifOrientation(data); got != 6 {
		t.Fatalf("expected orientation 6, got %d", got)
	}
	if got := exifOrientation([]byte{0xFF, 0xD8, 0xFF, 0xD9}); got != 1 {
		t.Fatalf("no EXIF should give 1, got %d", got)
	}
}
