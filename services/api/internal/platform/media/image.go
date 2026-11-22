package media

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"
)

const (
	MaxUploadBytes    = 10 << 20 // 10 MiB raw upload
	maxSourcePixels   = 40_000_000
	maxSourceSide     = 10_000
	minSourceSide     = 200
	maxOutputSide     = 1600
	outputJPEGQuality = 85
)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrImageTooSmall   = errors.New("image too small")
	ErrImageTooLarge   = errors.New("image too large")
	ErrCorruptImage    = errors.New("corrupt image")
)

// ProcessedImage is a sanitised JPEG ready to be stored.
type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}

// ProcessUpload validates an untrusted upload by content sniffing (never by
// filename or client-declared type), guards against decompression bombs,
// downsizes large images and re-encodes to JPEG. Re-encoding strips every
// metadata block, including EXIF GPS coordinates.
func ProcessUpload(raw []byte) (ProcessedImage, error) {
	if len(raw) == 0 {
		return ProcessedImage{}, ErrCorruptImage
	}
	if len(raw) > MaxUploadBytes {
		return ProcessedImage{}, ErrImageTooLarge
	}
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png":
	default:
		return ProcessedImage{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}
	if cfg.Width < minSourceSide || cfg.Height < minSourceSide {
		return ProcessedImage{}, ErrImageTooSmall
	}
	if cfg.Width > maxSourceSide || cfg.Height > maxSourceSide || cfg.Width*cfg.Height > maxSourcePixels {
		return ProcessedImage{}, ErrImageTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrCorruptImage
	}
	out := downscale(src, maxOutputSide)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: outputJPEGQuality}); err != nil {
		return ProcessedImage{}, err
	}
	b := out.Bounds()
	return ProcessedImage{Data: buf.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

// downscale converts to RGBA (flattening transparency on white) and
// box-filters the image so its longest side is at most maxSide.
func downscale(src image.Image, maxSide int) *image.RGBA {
	sb := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, sb.Dx(), sb.Dy()))
	draw.Draw(rgba, rgba.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(rgba, rgba.Bounds(), src, sb.Min, draw.Over)

	w, h := sb.Dx(), sb.Dy()
	longest := w
	if h > longest {
		longest = h
	}
	if longest <= maxSide {
		return rgba
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
			var r, g, b, a, n uint32
			for sy := sy0; sy < sy1; sy++ {
				row := sy * rgba.Stride
				for sx := sx0; sx < sx1; sx++ {
					i := row + sx*4
					r += uint32(rgba.Pix[i])
					g += uint32(rgba.Pix[i+1])
					b += uint32(rgba.Pix[i+2])
					a += uint32(rgba.Pix[i+3])
					n++
				}
			}
			o := y*dst.Stride + x*4
			dst.Pix[o] = uint8(r / n)
			dst.Pix[o+1] = uint8(g / n)
			dst.Pix[o+2] = uint8(b / n)
			dst.Pix[o+3] = uint8(a / n)
		}
	}
	return dst
}
