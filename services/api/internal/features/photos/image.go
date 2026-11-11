package photos

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrImageTooSmall   = errors.New("image too small")
	ErrImageTooLarge   = errors.New("image dimensions too large")
	ErrCorruptImage    = errors.New("corrupt image")
)

const (
	MaxUploadBytes = 10 << 20 // 10 MiB
	minDimension   = 200
	maxDimension   = 10000
	maxPixels      = 40_000_000
	maxOutputEdge  = 1600
	jpegQuality    = 82
)

type Processed struct {
	Data   []byte
	Width  int
	Height int
}

// Process validates an uploaded image by its content (never by file name or
// client-declared type), guards against decompression bombs, downsizes it and
// re-encodes it as JPEG. Re-encoding drops every metadata block (EXIF GPS
// coordinates, device info) and any payload appended to the file.
func Process(raw []byte) (Processed, error) {
	decodeConfig, decode, err := codecFor(http.DetectContentType(raw))
	if err != nil {
		return Processed{}, err
	}
	cfg, err := decodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorruptImage
	}
	if cfg.Width < minDimension || cfg.Height < minDimension {
		return Processed{}, ErrImageTooSmall
	}
	if cfg.Width > maxDimension || cfg.Height > maxDimension || cfg.Width*cfg.Height > maxPixels {
		return Processed{}, ErrImageTooLarge
	}
	src, err := decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorruptImage
	}

	w, h := fit(src.Bounds().Dx(), src.Bounds().Dy(), maxOutputEdge)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	// Flatten transparency on white before JPEG encoding.
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Processed{}, err
	}
	return Processed{Data: out.Bytes(), Width: w, Height: h}, nil
}

type configFunc func(r *bytes.Reader) (image.Config, error)
type decodeFunc func(r *bytes.Reader) (image.Image, error)

func codecFor(contentType string) (configFunc, decodeFunc, error) {
	switch contentType {
	case "image/jpeg":
		return func(r *bytes.Reader) (image.Config, error) { return jpeg.DecodeConfig(r) },
			func(r *bytes.Reader) (image.Image, error) { return jpeg.Decode(r) }, nil
	case "image/png":
		return func(r *bytes.Reader) (image.Config, error) { return png.DecodeConfig(r) },
			func(r *bytes.Reader) (image.Image, error) { return png.Decode(r) }, nil
	case "image/webp":
		return func(r *bytes.Reader) (image.Config, error) { return webp.DecodeConfig(r) },
			func(r *bytes.Reader) (image.Image, error) { return webp.Decode(r) }, nil
	default:
		return nil, nil, ErrUnsupportedType
	}
}

func fit(w, h, maxEdge int) (int, int) {
	if w <= maxEdge && h <= maxEdge {
		return w, h
	}
	if w >= h {
		return maxEdge, max(1, h*maxEdge/w)
	}
	return max(1, w*maxEdge/h), maxEdge
}
