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
	ErrUnsupportedType = errors.New("only JPEG, PNG and WebP images are accepted")
	ErrImageTooLarge   = errors.New("image is too large")
	ErrInvalidImage    = errors.New("file is not a valid image")
	ErrImageTooSmall   = errors.New("image must be at least 200x200 pixels")
)

// ProcessImage validates an uploaded image by sniffing its real content type
// (the client-provided type is ignored), checks dimensions before decoding to
// defeat decompression bombs, then decodes, downsizes and re-encodes it as
// JPEG. Re-encoding strips every metadata block, including EXIF GPS data.
func ProcessImage(raw []byte) (ProcessedImage, error) {
	if len(raw) == 0 {
		return ProcessedImage{}, ErrInvalidImage
	}
	if len(raw) > MaxUploadBytes {
		return ProcessedImage{}, ErrImageTooLarge
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
	case "image/webp":
		decodeConfig = func(b []byte) (image.Config, error) { return webp.DecodeConfig(bytes.NewReader(b)) }
		decode = func(b []byte) (image.Image, error) { return webp.Decode(bytes.NewReader(b)) }
	default:
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, err := decodeConfig(raw)
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > MaxSourcePixels {
		return ProcessedImage{}, ErrImageTooLarge
	}
	if cfg.Width < 200 || cfg.Height < 200 {
		return ProcessedImage{}, ErrImageTooSmall
	}

	src, err := decode(raw)
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w > MaxDimension || h > MaxDimension {
		if w >= h {
			h = h * MaxDimension / w
			w = MaxDimension
		} else {
			w = w * MaxDimension / h
			h = MaxDimension
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	// Flatten transparency on white, then scale.
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return ProcessedImage{}, err
	}
	return ProcessedImage{Data: out.Bytes(), Width: w, Height: h}, nil
}
