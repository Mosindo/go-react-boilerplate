package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"io"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

var allowedTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

// Processed is a sanitised JPEG ready to be stored.
type Processed struct {
	Data   []byte
	Width  int
	Height int
}

// Process validates untrusted upload bytes and re-encodes them as a bounded
// JPEG. Decoding + re-encoding drops EXIF (GPS!), embedded payloads and
// polyglot tricks; the declared filename and content type are never trusted.
func Process(r io.Reader) (Processed, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxUploadSize+1))
	if err != nil {
		return Processed{}, platformerrors.Invalid("could not read upload")
	}
	if len(raw) > MaxUploadSize {
		return Processed{}, platformerrors.New(http.StatusRequestEntityTooLarge, "file_too_large", "photo must be 8 MB or smaller")
	}
	if len(raw) == 0 {
		return Processed{}, platformerrors.Invalid("empty file")
	}
	if !allowedTypes[http.DetectContentType(raw)] {
		return Processed{}, platformerrors.New(http.StatusUnsupportedMediaType, "unsupported_type", "only JPEG, PNG or WebP images are accepted")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 {
		return Processed{}, platformerrors.Invalid("invalid image")
	}
	if cfg.Width*cfg.Height > maxPixels {
		return Processed{}, platformerrors.Invalid("image dimensions are too large")
	}
	if cfg.Width < 200 || cfg.Height < 200 {
		return Processed{}, platformerrors.Invalid("image must be at least 200x200 pixels")
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, platformerrors.Invalid("invalid image")
	}

	w, h := cfg.Width, cfg.Height
	if w > MaxDimension || h > MaxDimension {
		if w >= h {
			h, w = h*MaxDimension/w, MaxDimension
		} else {
			w, h = w*MaxDimension/h, MaxDimension
		}
	}
	// Flatten onto white: removes alpha and normalises every colour model.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 82}); err != nil {
		return Processed{}, err
	}
	return Processed{Data: out.Bytes(), Width: w, Height: h}, nil
}
