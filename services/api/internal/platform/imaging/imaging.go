// Package imaging validates and normalizes user-uploaded images.
//
// Every upload is fully decoded and re-encoded as JPEG. This strips all
// metadata (EXIF, including GPS position), neutralizes polyglot files and
// guarantees the stored bytes are a plain image.
package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

const (
	MaxUploadBytes = 10 << 20
	MinDimension   = 200
	MaxDimension   = 8000
	MaxPixels      = 40_000_000
	OutputMaxSide  = 1600
	jpegQuality    = 85
)

var (
	ErrUnsupportedType = errors.New("only JPEG, PNG and WebP images are accepted")
	ErrTooSmall        = errors.New("image must be at least 200x200 pixels")
	ErrTooLarge        = errors.New("image dimensions are too large")
	ErrCorrupt         = errors.New("image could not be decoded")
)

var allowedTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

type Result struct {
	Data   []byte
	Width  int
	Height int
}

// Process validates raw bytes and returns a normalized JPEG.
func Process(raw []byte) (Result, error) {
	if len(raw) == 0 || len(raw) > MaxUploadBytes {
		return Result{}, ErrTooLarge
	}
	// Content sniffing on magic bytes: the client-declared type is ignored.
	if !allowedTypes[http.DetectContentType(raw)] {
		return Result{}, ErrUnsupportedType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Result{}, ErrCorrupt
	}
	if cfg.Width < MinDimension || cfg.Height < MinDimension {
		return Result{}, ErrTooSmall
	}
	// Checked before decoding to avoid decompression bombs.
	if cfg.Width > MaxDimension || cfg.Height > MaxDimension || cfg.Width*cfg.Height > MaxPixels {
		return Result{}, ErrTooLarge
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Result{}, ErrCorrupt
	}

	w, h := fit(src.Bounds().Dx(), src.Bounds().Dy(), OutputMaxSide)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Result{}, err
	}
	return Result{Data: buf.Bytes(), Width: w, Height: h}, nil
}

func fit(w, h, maxSide int) (int, int) {
	if w <= maxSide && h <= maxSide {
		return w, h
	}
	if w >= h {
		return maxSide, max(1, h*maxSide/w)
	}
	return max(1, w*maxSide/h), maxSide
}
