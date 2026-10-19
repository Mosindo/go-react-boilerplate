package profiles

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder
	"net/http"
)

const (
	MaxUploadBytes = 8 << 20
	maxPixels      = 40_000_000
	maxDimension   = 10_000
	minDimension   = 200
	outputMaxSide  = 1280
	jpegQuality    = 82
)

// ProcessedImage is the sanitised, re-encoded photo that gets stored.
type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}

// ProcessImage validates an uploaded file and returns a freshly encoded JPEG.
// Re-encoding (rather than storing the upload) drops EXIF/GPS metadata and any
// payload smuggled in trailing bytes, and the pixel budget is checked from the
// header before decoding to defuse decompression bombs.
func ProcessImage(raw []byte) (ProcessedImage, error) {
	if len(raw) == 0 {
		return ProcessedImage{}, ErrInvalidImage
	}
	if len(raw) > MaxUploadBytes {
		return ProcessedImage{}, ErrImageTooLarge
	}
	sniff := raw
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	switch http.DetectContentType(sniff) {
	case "image/jpeg", "image/png":
	default:
		return ProcessedImage{}, ErrInvalidImage
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}
	if cfg.Width < minDimension || cfg.Height < minDimension ||
		cfg.Width > maxDimension || cfg.Height > maxDimension || cfg.Width*cfg.Height > maxPixels {
		return ProcessedImage{}, ErrInvalidImage
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}

	rgba := flatten(img)
	rgba = downscale(rgba, outputMaxSide)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, rgba, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return ProcessedImage{}, ErrInvalidImage
	}
	b := rgba.Bounds()
	return ProcessedImage{Data: out.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

// flatten draws the image over white so transparent PNGs become opaque.
func flatten(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

// downscale shrinks the image so its longest side is at most maxSide, using
// area averaging (good quality for photos, no extra dependency).
func downscale(src *image.RGBA, maxSide int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	longest := w
	if h > longest {
		longest = h
	}
	if longest <= maxSide {
		return src
	}
	nw := w * maxSide / longest
	nh := h * maxSide / longest
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy0, sy1 := y*h/nh, (y+1)*h/nh
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < nw; x++ {
			sx0, sx1 := x*w/nw, (x+1)*w/nw
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, bl, n uint32
			for sy := sy0; sy < sy1; sy++ {
				row := src.Pix[sy*src.Stride:]
				for sx := sx0; sx < sx1; sx++ {
					i := sx * 4
					r += uint32(row[i])
					g += uint32(row[i+1])
					bl += uint32(row[i+2])
					n++
				}
			}
			o := y*dst.Stride + x*4
			dst.Pix[o] = uint8(r / n)
			dst.Pix[o+1] = uint8(g / n)
			dst.Pix[o+2] = uint8(bl / n)
			dst.Pix[o+3] = 255
		}
	}
	return dst
}
