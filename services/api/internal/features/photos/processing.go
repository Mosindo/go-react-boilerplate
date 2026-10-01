package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"

	xdraw "golang.org/x/image/draw"
)

var (
	ErrUnsupportedType = errors.New("only JPEG and PNG images are accepted")
	ErrImageTooLarge   = errors.New("image dimensions too large")
	ErrImageTooSmall   = errors.New("image must be at least 200 px on each side")
	ErrInvalidImage    = errors.New("invalid image")
)

type processed struct {
	Data   []byte
	Width  int
	Height int
}

// processImage validates untrusted bytes and re-encodes them as a clean JPEG.
// Re-encoding drops EXIF/GPS metadata and any payload appended to the file;
// DecodeConfig runs first so decompression bombs are rejected before decoding.
func processImage(raw []byte) (processed, error) {
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png":
	default:
		return processed{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return processed{}, ErrInvalidImage
	}
	if cfg.Width > MaxSidePx || cfg.Height > MaxSidePx || cfg.Width*cfg.Height > MaxPixels {
		return processed{}, ErrImageTooLarge
	}
	if cfg.Width < MinSidePx || cfg.Height < MinSidePx {
		return processed{}, ErrImageTooSmall
	}

	var src image.Image
	if http.DetectContentType(raw) == "image/png" {
		src, err = png.Decode(bytes.NewReader(raw))
	} else {
		src, err = jpeg.Decode(bytes.NewReader(raw))
	}
	if err != nil {
		return processed{}, ErrInvalidImage
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if longest := max(w, h); longest > OutputMaxSidePx {
		scale := float64(OutputMaxSidePx) / float64(longest)
		dw, dh = int(float64(w)*scale+0.5), int(float64(h)*scale+0.5)
	}

	// Flatten onto white so transparent PNGs do not turn black in JPEG.
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	if dw == w && dh == h {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	} else {
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: OutputJPEGQuality}); err != nil {
		return processed{}, err
	}
	return processed{Data: out.Bytes(), Width: dw, Height: dh}, nil
}
