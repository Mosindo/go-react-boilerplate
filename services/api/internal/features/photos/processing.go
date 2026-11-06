package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder
	"net/http"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder with image.Decode
)

var (
	ErrUnsupportedType = errors.New("unsupported image type: use JPEG, PNG or WebP")
	ErrCorruptImage    = errors.New("the image could not be read")
	ErrImageTooLarge   = errors.New("image dimensions are too large")
	ErrImageTooSmall   = errors.New("image is too small (minimum 200 px)")
)

// Processed is a sanitised, re-encoded image ready to store.
type Processed struct {
	Data          []byte
	Width, Height int
}

// Process validates untrusted upload bytes and re-encodes them as a JPEG.
//
// The content type is sniffed from the bytes (the client-declared type is
// ignored), dimensions are bounded before any pixel decoding, and the image is
// re-encoded: this drops EXIF/GPS metadata and any payload hidden in the file.
func Process(raw []byte) (Processed, error) {
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return Processed{}, ErrUnsupportedType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorruptImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return Processed{}, ErrCorruptImage
	}
	if int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return Processed{}, ErrImageTooLarge
	}
	if cfg.Width < minSide || cfg.Height < minSide {
		return Processed{}, ErrImageTooSmall
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorruptImage
	}

	w, h := fit(src.Bounds().Dx(), src.Bounds().Dy(), MaxSide)
	// Flatten onto white: JPEG has no alpha channel.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return Processed{}, err
	}
	return Processed{Data: out.Bytes(), Width: w, Height: h}, nil
}

// fit scales (w,h) down so the longest side is at most max, preserving ratio.
func fit(w, h, max int) (int, int) {
	if w <= max && h <= max {
		return w, h
	}
	if w >= h {
		return max, maxInt(1, h*max/w)
	}
	return maxInt(1, w*max/h), max
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
