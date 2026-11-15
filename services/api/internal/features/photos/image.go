package photos

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"

	"example.com/api/internal/platform/apperr"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

var (
	ErrUnsupportedType = apperr.BadRequest("unsupported_image_type", "only JPEG, PNG or WebP images are accepted")
	ErrImageTooLarge   = apperr.New(http.StatusRequestEntityTooLarge, "image_too_large", "image must be smaller than 10 MB")
	ErrImageDimensions = apperr.BadRequest("invalid_image_dimensions", "image must be at least 200x200 pixels")
	ErrCorruptImage    = apperr.BadRequest("invalid_image", "the file is not a valid image")
)

var allowedTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// ProcessImage validates an upload by its content (never the client-supplied
// name or MIME type), bounds its dimensions before decoding, downscales it and
// re-encodes it as JPEG. Re-encoding drops every metadata block, including
// EXIF GPS coordinates, and neutralises polyglot or malformed files.
func ProcessImage(data []byte) (ProcessedImage, error) {
	if len(data) == 0 {
		return ProcessedImage{}, ErrCorruptImage
	}
	if len(data) > MaxUploadBytes {
		return ProcessedImage{}, ErrImageTooLarge
	}
	if !allowedTypes[http.DetectContentType(data)] {
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png" && format != "webp") {
		return ProcessedImage{}, ErrCorruptImage
	}
	if cfg.Width < MinDimension || cfg.Height < MinDimension {
		return ProcessedImage{}, ErrImageDimensions
	}
	if cfg.Width > MaxDimension || cfg.Height > MaxDimension || cfg.Width*cfg.Height > MaxSourcePixels {
		return ProcessedImage{}, ErrImageTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}

	w, h := FitWithin(src.Bounds().Dx(), src.Bounds().Dy(), OutputMaxSide)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	// Paint white first so transparent PNG areas do not turn black in JPEG.
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: OutputJPEGQuality}); err != nil {
		return ProcessedImage{}, err
	}
	return ProcessedImage{Data: out.Bytes(), Width: w, Height: h}, nil
}

// FitWithin scales (w, h) down so the longest side is at most maxSide.
func FitWithin(w, h, maxSide int) (int, int) {
	if w <= maxSide && h <= maxSide {
		return w, h
	}
	if w >= h {
		return maxSide, max(1, h*maxSide/w)
	}
	return max(1, w*maxSide/h), maxSide
}
