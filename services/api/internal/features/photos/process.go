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
	ErrImageTooSmall   = errors.New("image is too small (min 200x200)")
	ErrImageTooLarge   = errors.New("image dimensions are too large")
	ErrCorruptImage    = errors.New("image could not be decoded")
)

var allowedTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true}

// Processed is a sanitized image ready to be stored.
type Processed struct {
	Data   []byte
	Width  int
	Height int
}

// Process validates an untrusted upload and re-encodes it.
//
// Security properties:
//   - the type is sniffed from the bytes (client-provided MIME/extension are ignored);
//   - dimensions are checked from the header before full decoding (decompression bombs);
//   - the output is a freshly encoded JPEG: EXIF (including GPS position),
//     ICC/XMP metadata and any trailing/polyglot payload are dropped;
//   - the image is downscaled to MaxOutputSide to bound storage and bandwidth.
func Process(raw []byte) (Processed, error) {
	if !allowedTypes[http.DetectContentType(raw)] {
		return Processed{}, ErrUnsupportedType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorruptImage
	}
	if cfg.Width < MinSourceSide || cfg.Height < MinSourceSide {
		return Processed{}, ErrImageTooSmall
	}
	if cfg.Width*cfg.Height > MaxSourcePixels || cfg.Width > 12000 || cfg.Height > 12000 {
		return Processed{}, ErrImageTooLarge
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorruptImage
	}

	w, h := fit(src.Bounds().Dx(), src.Bounds().Dy(), MaxOutputSide)
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return Processed{}, err
	}
	return Processed{Data: buf.Bytes(), Width: w, Height: h}, nil
}

func fit(w, h, max int) (int, int) {
	if w <= max && h <= max {
		return w, h
	}
	if w >= h {
		return max, max * h / w
	}
	return max * w / h, max
}
