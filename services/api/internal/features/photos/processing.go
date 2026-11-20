package photos

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

var (
	ErrUnsupportedType = errors.New("only JPEG, PNG and WebP images are accepted")
	ErrInvalidImage    = errors.New("the file is not a valid image")
	ErrImageTooSmall   = errors.New("image must be at least 200x200 pixels")
	ErrImageTooLarge   = errors.New("image dimensions are too large")
)

var allowedTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

type processedImage struct {
	data   []byte
	width  int
	height int
}

// processImage validates an upload by its magic bytes (never by the client's
// declared type or extension), guards against decompression bombs, then
// re-encodes it as a resized JPEG. Re-encoding strips all metadata, including
// EXIF GPS coordinates, and neutralises polyglot or malformed files.
func processImage(raw []byte) (processedImage, error) {
	if !allowedTypes[http.DetectContentType(raw)] {
		return processedImage{}, ErrUnsupportedType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return processedImage{}, ErrInvalidImage
	}
	if cfg.Width > maxSourceSide || cfg.Height > maxSourceSide || cfg.Width*cfg.Height > maxSourcePixels {
		return processedImage{}, ErrImageTooLarge
	}
	if cfg.Width < minSourceSide || cfg.Height < minSourceSide {
		return processedImage{}, ErrImageTooSmall
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return processedImage{}, ErrInvalidImage
	}

	width, height := fitWithin(src.Bounds().Dx(), src.Bounds().Dy(), maxOutputSide)
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return processedImage{}, err
	}
	return processedImage{data: buf.Bytes(), width: width, height: height}, nil
}

func fitWithin(width, height, maxSide int) (int, int) {
	if width <= maxSide && height <= maxSide {
		return width, height
	}
	if width >= height {
		return maxSide, max(1, height*maxSide/width)
	}
	return max(1, width*maxSide/height), maxSide
}
