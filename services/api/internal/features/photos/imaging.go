package photos

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"
)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrImageTooSmall   = errors.New("image too small")
	ErrImageTooLarge   = errors.New("image dimensions too large")
	ErrCorruptImage    = errors.New("corrupt image")
)

const (
	MaxUploadBytes  = 10 << 20 // 10 MiB
	minSidePx       = 200
	maxSidePx       = 8000
	maxPixels       = 40_000_000
	targetLongSide  = 1600
	jpegQuality     = 82
	outputMediaType = "image/jpeg"
)

type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}

// ProcessImage validates an upload by magic bytes (never by client-provided headers),
// rejects decompression bombs before decoding, downsizes large pictures and re-encodes
// them as JPEG. Re-encoding drops all metadata, including EXIF GPS coordinates.
func ProcessImage(data []byte) (ProcessedImage, error) {
	sniffed := http.DetectContentType(data)
	var decodeConfig func([]byte) (image.Config, error)
	var decode func([]byte) (image.Image, error)
	switch sniffed {
	case "image/jpeg":
		decodeConfig = func(b []byte) (image.Config, error) { return jpeg.DecodeConfig(bytes.NewReader(b)) }
		decode = func(b []byte) (image.Image, error) { return jpeg.Decode(bytes.NewReader(b)) }
	case "image/png":
		decodeConfig = func(b []byte) (image.Config, error) { return png.DecodeConfig(bytes.NewReader(b)) }
		decode = func(b []byte) (image.Image, error) { return png.Decode(bytes.NewReader(b)) }
	default:
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, err := decodeConfig(data)
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}
	if cfg.Width < minSidePx || cfg.Height < minSidePx {
		return ProcessedImage{}, ErrImageTooSmall
	}
	if cfg.Width > maxSidePx || cfg.Height > maxSidePx || cfg.Width*cfg.Height > maxPixels {
		return ProcessedImage{}, ErrImageTooLarge
	}

	src, err := decode(data)
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}

	// Flatten onto an opaque canvas (transparent PNG areas become white).
	bounds := src.Bounds()
	canvas := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), src, bounds.Min, draw.Over)

	out := downscale(canvas, targetLongSide)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return ProcessedImage{}, err
	}
	return ProcessedImage{Data: buf.Bytes(), Width: out.Bounds().Dx(), Height: out.Bounds().Dy()}, nil
}

// downscale shrinks img so its longest side is at most maxSide using area averaging.
func downscale(img *image.RGBA, maxSide int) *image.RGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	longest := w
	if h > longest {
		longest = h
	}
	if longest <= maxSide {
		return img
	}
	dw := w * maxSide / longest
	dh := h * maxSide / longest
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		sy0 := y * h / dh
		sy1 := (y + 1) * h / dh
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < dw; x++ {
			sx0 := x * w / dw
			sx1 := (x + 1) * w / dw
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, b, n uint32
			for sy := sy0; sy < sy1; sy++ {
				row := img.Pix[sy*img.Stride:]
				for sx := sx0; sx < sx1; sx++ {
					i := sx * 4
					r += uint32(row[i])
					g += uint32(row[i+1])
					b += uint32(row[i+2])
					n++
				}
			}
			o := dst.PixOffset(x, y)
			dst.Pix[o] = uint8(r / n)
			dst.Pix[o+1] = uint8(g / n)
			dst.Pix[o+2] = uint8(b / n)
			dst.Pix[o+3] = 255
		}
	}
	return dst
}
