package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
)

var (
	ErrUnsupportedType = platformerrors.New(http.StatusUnsupportedMediaType, "only JPEG and PNG images are accepted")
	ErrInvalidImage    = platformerrors.Invalid("the file is not a valid image")
	ErrImageTooSmall   = platformerrors.Invalid("image is too small (minimum 200x200 pixels)")
	ErrImageTooLarge   = platformerrors.Invalid("image dimensions are too large")
)

// processImage validates raw upload bytes and returns a sanitised JPEG:
//   - the type is detected from the content (never from the client-provided name or header),
//   - dimensions are checked before decoding to avoid decompression bombs,
//   - the image is re-encoded, which drops EXIF/GPS and any appended payload,
//   - large images are downscaled.
func processImage(raw []byte) (out []byte, width, height int, err error) {
	if len(raw) == 0 {
		return nil, 0, 0, ErrInvalidImage
	}
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png":
	default:
		return nil, 0, 0, ErrUnsupportedType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, ErrInvalidImage
	}
	if cfg.Width < MinSide || cfg.Height < MinSide {
		return nil, 0, 0, ErrImageTooSmall
	}
	if cfg.Width*cfg.Height > MaxPixels {
		return nil, 0, 0, ErrImageTooLarge
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, 0, 0, ErrInvalidImage
	}
	img = downscale(img, MaxOutputSide)
	b := img.Bounds()

	// Flatten transparency on white: JPEG has no alpha channel.
	flat := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(flat, flat.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(flat, flat.Bounds(), img, b.Min, draw.Over)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, flat, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), b.Dx(), b.Dy(), nil
}

// downscale shrinks img so that its longest side is at most maxSide (bilinear sampling).
func downscale(src image.Image, maxSide int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxSide && h <= maxSide {
		return src
	}
	nw, nh := maxSide, maxSide
	if w >= h {
		nh = max(1, h*maxSide/w)
	} else {
		nw = max(1, w*maxSide/h)
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xr, yr := float64(w)/float64(nw), float64(h)/float64(nh)
	for y := 0; y < nh; y++ {
		sy := (float64(y)+0.5)*yr - 0.5
		y0 := clamp(int(sy), 0, h-1)
		y1 := clamp(y0+1, 0, h-1)
		fy := sy - float64(y0)
		if fy < 0 {
			fy = 0
		}
		for x := 0; x < nw; x++ {
			sx := (float64(x)+0.5)*xr - 0.5
			x0 := clamp(int(sx), 0, w-1)
			x1 := clamp(x0+1, 0, w-1)
			fx := sx - float64(x0)
			if fx < 0 {
				fx = 0
			}
			r00, g00, b00, a00 := src.At(b.Min.X+x0, b.Min.Y+y0).RGBA()
			r10, g10, b10, a10 := src.At(b.Min.X+x1, b.Min.Y+y0).RGBA()
			r01, g01, b01, a01 := src.At(b.Min.X+x0, b.Min.Y+y1).RGBA()
			r11, g11, b11, a11 := src.At(b.Min.X+x1, b.Min.Y+y1).RGBA()
			mix := func(c00, c10, c01, c11 uint32) uint8 {
				top := float64(c00)*(1-fx) + float64(c10)*fx
				bot := float64(c01)*(1-fx) + float64(c11)*fx
				return uint8((top*(1-fy) + bot*fy) / 257)
			}
			dst.SetRGBA(x, y, color.RGBA{mix(r00, r10, r01, r11), mix(g00, g10, g01, g11), mix(b00, b10, b01, b11), mix(a00, a10, a01, a11)})
		}
	}
	return dst
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
