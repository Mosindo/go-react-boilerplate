package media

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"io"
	"net/http"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

const (
	MaxUploadBytes = 10 << 20
	MaxDimension   = 1600
	MinDimension   = 200
	maxSourcePixel = 24_000_000
	maxSourceSide  = 12_000
	jpegQuality    = 85
)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrImageTooLarge   = errors.New("image too large")
	ErrImageTooSmall   = errors.New("image too small")
	ErrInvalidImage    = errors.New("invalid image")
)

// decodeSlots bounds concurrent full decodes (each can take ~100 MB of RAM).
var decodeSlots = make(chan struct{}, 3)

var allowedContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

// ProcessedImage is a sanitised JPEG ready to be stored.
type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}

// ProcessUpload validates an uploaded image by its content (never by the
// client supplied name or header), rejects decompression bombs, downsizes it
// and re-encodes it as JPEG. Re-encoding strips EXIF metadata such as GPS
// coordinates and neutralises polyglot files.
func ProcessUpload(r io.Reader) (ProcessedImage, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}
	if len(raw) > MaxUploadBytes {
		return ProcessedImage{}, ErrImageTooLarge
	}
	if _, ok := allowedContentTypes[sniff(raw)]; !ok {
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}
	if cfg.Width > maxSourceSide || cfg.Height > maxSourceSide || cfg.Width*cfg.Height > maxSourcePixel {
		return ProcessedImage{}, ErrImageTooLarge
	}
	if cfg.Width < MinDimension || cfg.Height < MinDimension {
		return ProcessedImage{}, ErrImageTooSmall
	}

	decodeSlots <- struct{}{}
	defer func() { <-decodeSlots }()

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}

	width, height := fitWithin(src.Bounds().Dx(), src.Bounds().Dy(), MaxDimension)
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return ProcessedImage{}, err
	}
	return ProcessedImage{Data: out.Bytes(), Width: width, Height: height}, nil
}

func sniff(raw []byte) string {
	// net/http does not detect WebP on every Go version; check the RIFF header.
	if len(raw) >= 12 && string(raw[0:4]) == "RIFF" && string(raw[8:12]) == "WEBP" {
		return "image/webp"
	}
	return http.DetectContentType(raw)
}

func fitWithin(width, height, max int) (int, int) {
	if width <= max && height <= max {
		return width, height
	}
	if width >= height {
		return max, maxInt(1, height*max/width)
	}
	return maxInt(1, width*max/height), max
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
