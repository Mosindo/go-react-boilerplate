package media

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"

	"golang.org/x/image/draw"
)

const (
	// MaxUploadBytes is the hard limit on raw uploads.
	MaxUploadBytes = 8 << 20
	// maxPixels guards against decompression bombs.
	maxPixels = 40_000_000
	// maxEdge is the longest edge of stored images.
	maxEdge = 1440
	// minEdge rejects images too small to be useful profile photos.
	minEdge     = 200
	jpegQuality = 82
)

var (
	ErrUnsupportedType = errors.New("only JPEG and PNG images are accepted")
	ErrTooLarge        = errors.New("image is too large")
	ErrTooSmall        = errors.New("image is too small (minimum 200x200)")
	ErrCorrupt         = errors.New("image could not be decoded")
)

type Processed struct {
	Data   []byte
	Width  int
	Height int
}

// Process validates an uploaded image by its content (never by client-declared
// metadata), downsizes it and re-encodes it as JPEG. Re-encoding strips every
// metadata block, including EXIF GPS coordinates.
func Process(raw []byte) (Processed, error) {
	if len(raw) == 0 {
		return Processed{}, ErrCorrupt
	}
	if len(raw) > MaxUploadBytes {
		return Processed{}, ErrTooLarge
	}

	var decodeConfig func([]byte) (image.Config, error)
	var decode func([]byte) (image.Image, error)
	switch http.DetectContentType(raw) {
	case "image/jpeg":
		decodeConfig = func(b []byte) (image.Config, error) { return jpeg.DecodeConfig(bytes.NewReader(b)) }
		decode = func(b []byte) (image.Image, error) { return jpeg.Decode(bytes.NewReader(b)) }
	case "image/png":
		decodeConfig = func(b []byte) (image.Config, error) { return png.DecodeConfig(bytes.NewReader(b)) }
		decode = func(b []byte) (image.Image, error) { return png.Decode(bytes.NewReader(b)) }
	default:
		return Processed{}, ErrUnsupportedType
	}

	cfg, err := decodeConfig(raw)
	if err != nil {
		return Processed{}, ErrCorrupt
	}
	if cfg.Width*cfg.Height > maxPixels {
		return Processed{}, ErrTooLarge
	}
	if cfg.Width < minEdge || cfg.Height < minEdge {
		return Processed{}, ErrTooSmall
	}

	src, err := decode(raw)
	if err != nil {
		return Processed{}, ErrCorrupt
	}

	width, height := cfg.Width, cfg.Height
	if width > maxEdge || height > maxEdge {
		if width >= height {
			height = height * maxEdge / width
			width = maxEdge
		} else {
			width = width * maxEdge / height
			height = maxEdge
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	// Flatten transparency onto white before JPEG encoding.
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Processed{}, err
	}
	return Processed{Data: out.Bytes(), Width: width, Height: height}, nil
}
