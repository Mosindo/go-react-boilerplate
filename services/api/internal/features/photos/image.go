package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"

	perr "example.com/api/internal/platform/errors"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

const (
	maxUploadBytes = 8 << 20
	minSide        = 200
	maxSide        = 12000
	maxPixels      = 50_000_000
	fullSide       = 1080
	thumbSide      = 360
)

type processed struct {
	Full, Thumb   []byte
	Width, Height int
}

// processImage validates an upload and re-encodes it. Re-encoding drops metadata (EXIF/GPS) and
// any payload hidden in the original file; only decoded pixels survive.
func processImage(raw []byte) (processed, error) {
	if len(raw) == 0 {
		return processed{}, perr.Invalid("empty_file", "no file received")
	}
	if len(raw) > maxUploadBytes {
		return processed{}, perr.Invalid("file_too_large", "photo must be at most 8 MB")
	}
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return processed{}, perr.Invalid("unsupported_type", "only JPEG, PNG and WebP photos are accepted")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return processed{}, perr.Invalid("invalid_image", "the file is not a valid image")
	}
	if cfg.Width < minSide || cfg.Height < minSide {
		return processed{}, perr.Invalid("image_too_small", "photo must be at least 200x200 pixels")
	}
	if cfg.Width > maxSide || cfg.Height > maxSide || cfg.Width*cfg.Height > maxPixels {
		return processed{}, perr.Invalid("image_too_large", "photo dimensions are too large")
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return processed{}, perr.Invalid("invalid_image", "the file is not a valid image")
	}

	full, w, h, err := encodeFit(src, fullSide, 82)
	if err != nil {
		return processed{}, err
	}
	thumb, _, _, err := encodeFit(src, thumbSide, 78)
	if err != nil {
		return processed{}, err
	}
	return processed{Full: full, Thumb: thumb, Width: w, Height: h}, nil
}

// encodeFit scales src to fit in a side x side box (never upscaling) and encodes it as JPEG,
// flattening transparency on white.
func encodeFit(src image.Image, side, quality int) ([]byte, int, int, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > side || h > side {
		if w >= h {
			h, w = h*side/w, side
		} else {
			w, h = w*side/h, side
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: quality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
}
