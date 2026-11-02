package photos

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	perr "example.com/api/internal/platform/errors"
)

const (
	MaxUploadBytes = 8 << 20 // raw upload limit
	MaxPixels      = 40_000_000
	MaxDimension   = 1080
	jpegQuality    = 82
)

var _ = gif.Decode // registers the GIF decoder (first frame only)
var _ = png.Decode // registers the PNG decoder

type Processed struct {
	Data          []byte
	Width, Height int
}

// Process validates an uploaded image by decoding it (content sniffing, not
// the client-supplied type), bounds its size, downscales it and re-encodes it
// as JPEG. Re-encoding drops EXIF/GPS metadata and any polyglot payload.
func Process(r io.Reader) (Processed, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return Processed{}, err
	}
	if len(raw) > MaxUploadBytes {
		return Processed{}, perr.New(413, "file_too_large", "La photo dépasse 8 Mo.")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "jpeg" && format != "png" && format != "gif") {
		return Processed{}, perr.BadRequest("invalid_image", "Format non supporté (JPEG, PNG ou GIF).")
	}
	if cfg.Width < 200 || cfg.Height < 200 {
		return Processed{}, perr.BadRequest("image_too_small", "La photo doit faire au moins 200×200 pixels.")
	}
	if cfg.Width*cfg.Height > MaxPixels {
		return Processed{}, perr.BadRequest("image_too_large", "Les dimensions de la photo sont trop grandes.")
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, perr.BadRequest("invalid_image", "Image illisible.")
	}
	img = downscale(img, MaxDimension)
	// Flatten transparency over white so JPEG output is well defined.
	b := img.Bounds()
	canvas := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), img, b.Min, draw.Over)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, canvas, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Processed{}, err
	}
	return Processed{Data: out.Bytes(), Width: canvas.Bounds().Dx(), Height: canvas.Bounds().Dy()}, nil
}

// downscale shrinks img so its longest side is at most max, using area
// averaging (good quality for photos, no extra dependency).
func downscale(src image.Image, max int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= max && h <= max {
		return src
	}
	var nw, nh int
	if w >= h {
		nw, nh = max, h*max/w
	} else {
		nh, nw = max, w*max/h
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy0 := b.Min.Y + y*h/nh
		sy1 := b.Min.Y + (y+1)*h/nh
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < nw; x++ {
			sx0 := b.Min.X + x*w/nw
			sx1 := b.Min.X + (x+1)*w/nw
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, bl, a, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					cr, cg, cb, ca := src.At(sx, sy).RGBA()
					r += uint64(cr)
					g += uint64(cg)
					bl += uint64(cb)
					a += uint64(ca)
					n++
				}
			}
			dst.SetRGBA(x, y, color.RGBA{uint8(r / n >> 8), uint8(g / n >> 8), uint8(bl / n >> 8), uint8(a / n >> 8)})
		}
	}
	return dst
}
