package photos

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"
)

// ProcessImage validates raw upload bytes and returns a sanitized JPEG.
// The image is fully decoded and re-encoded, which strips EXIF/GPS metadata and any
// non-image payload (polyglots), and downscaled to MaxDimension.
func ProcessImage(raw []byte) (data []byte, width, height int, err error) {
	if len(raw) == 0 || len(raw) > MaxUploadSize {
		return nil, 0, 0, fmt.Errorf("%w: size", ErrInvalidImage)
	}
	sniff := raw
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	if ct := http.DetectContentType(sniff); ct != "image/jpeg" && ct != "image/png" {
		return nil, 0, 0, fmt.Errorf("%w: only JPEG and PNG are accepted", ErrInvalidImage)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	if cfg.Width < 64 || cfg.Height < 64 {
		return nil, 0, 0, fmt.Errorf("%w: image too small", ErrInvalidImage)
	}
	if cfg.Width*cfg.Height > MaxPixels {
		return nil, 0, 0, fmt.Errorf("%w: image too large", ErrInvalidImage)
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("%w: %v", ErrInvalidImage, err)
	}
	img = downscale(img, MaxDimension)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, flatten(img), &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return nil, 0, 0, err
	}
	b := img.Bounds()
	return buf.Bytes(), b.Dx(), b.Dy(), nil
}

// flatten composites transparent pixels onto white so PNG alpha does not become black in JPEG.
func flatten(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			r, g, bl, a := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			// premultiplied -> over white
			inv := 0xffff - a
			dst.SetRGBA(x, y, color.RGBA{
				R: uint8((r + inv) >> 8), G: uint8((g + inv) >> 8), B: uint8((bl + inv) >> 8), A: 255,
			})
		}
	}
	return dst
}

// downscale shrinks so the longest side is at most max, using box averaging.
func downscale(src image.Image, max int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return src
	}
	nw, nh := max, max
	if w >= h {
		nh = h * max / w
	} else {
		nw = w * max / h
	}
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
			var r, g, bl, a, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					pr, pg, pb, pa := src.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
					r += uint64(pr)
					g += uint64(pg)
					bl += uint64(pb)
					a += uint64(pa)
					n++
				}
			}
			dst.SetRGBA(x, y, color.RGBA{uint8(r / n >> 8), uint8(g / n >> 8), uint8(bl / n >> 8), uint8(a / n >> 8)})
		}
	}
	return dst
}
