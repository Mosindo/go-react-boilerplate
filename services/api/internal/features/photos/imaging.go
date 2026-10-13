package photos

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder
	"io"
	"net/http"
)

var (
	ErrUnsupportedType = errors.New("only JPEG and PNG images are accepted")
	ErrTooLarge        = errors.New("image is too large")
	ErrTooSmall        = errors.New("image is too small")
	ErrCorrupt         = errors.New("image could not be read")
)

// Process validates an untrusted upload and returns re-encoded JPEG variants.
// Re-encoding drops metadata (EXIF, GPS) and any payload hidden in the file.
func Process(r io.Reader) (Processed, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return Processed{}, ErrCorrupt
	}
	if len(raw) > MaxUploadBytes {
		return Processed{}, ErrTooLarge
	}
	if len(raw) == 0 {
		return Processed{}, ErrCorrupt
	}
	if ct := http.DetectContentType(raw); ct != "image/jpeg" && ct != "image/png" {
		return Processed{}, ErrUnsupportedType
	}

	// Check dimensions from the header before decoding, to defuse decompression bombs.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorrupt
	}
	if cfg.Width < MinSide || cfg.Height < MinSide {
		return Processed{}, ErrTooSmall
	}
	if cfg.Width > MaxSourceSide || cfg.Height > MaxSourceSide || cfg.Width*cfg.Height > MaxSourcePixels {
		return Processed{}, ErrTooLarge
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorrupt
	}

	full := resizeToFit(flatten(src), FullMaxSide)
	thumb := resizeToFit(full, ThumbMaxSide)

	fullBytes, err := encodeJPEG(full, 82)
	if err != nil {
		return Processed{}, fmt.Errorf("encode photo: %w", err)
	}
	thumbBytes, err := encodeJPEG(thumb, 75)
	if err != nil {
		return Processed{}, fmt.Errorf("encode thumbnail: %w", err)
	}
	b := full.Bounds()
	return Processed{Width: b.Dx(), Height: b.Dy(), Data: fullBytes, Thumbnail: thumbBytes}, nil
}

func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// flatten draws the image onto an opaque white RGBA canvas (PNG transparency
// has no meaning in JPEG).
func flatten(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

// resizeToFit downsamples with area averaging so that the longest side is at
// most maxSide. Images already small enough are returned unchanged.
func resizeToFit(src *image.RGBA, maxSide int) *image.RGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw <= maxSide && sh <= maxSide {
		return src
	}
	var dw, dh int
	if sw >= sh {
		dw = maxSide
		dh = max(1, sh*maxSide/sw)
	} else {
		dh = maxSide
		dw = max(1, sw*maxSide/sh)
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		sy0, sy1 := y*sh/dh, (y+1)*sh/dh
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < dw; x++ {
			sx0, sx1 := x*sw/dw, (x+1)*sw/dw
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, b, n uint32
			for sy := sy0; sy < sy1; sy++ {
				row := src.Pix[sy*src.Stride:]
				for sx := sx0; sx < sx1; sx++ {
					i := sx * 4
					r += uint32(row[i])
					g += uint32(row[i+1])
					b += uint32(row[i+2])
					n++
				}
			}
			o := y*dst.Stride + x*4
			dst.Pix[o] = uint8(r / n)
			dst.Pix[o+1] = uint8(g / n)
			dst.Pix[o+2] = uint8(b / n)
			dst.Pix[o+3] = 255
		}
	}
	return dst
}
