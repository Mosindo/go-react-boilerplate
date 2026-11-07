package profiles

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder
)

const (
	MaxUploadBytes = 5 << 20
	maxSourcePixel = 8000 // reject decompression bombs before decoding pixels
	minSourceEdge  = 200
	maxOutputEdge  = 1280
	jpegQuality    = 82
)

var (
	ErrUnsupportedImage = errors.New("unsupported image: use JPEG or PNG")
	ErrImageTooSmall    = errors.New("image is too small (min 200px)")
	ErrImageTooLarge    = errors.New("image dimensions are too large")
)

type ProcessedImage struct {
	Data          []byte
	Width, Height int
}

// ProcessImage validates untrusted bytes and re-encodes them as a clean, downscaled JPEG.
// Re-encoding strips EXIF/GPS metadata and any polyglot payload.
func ProcessImage(raw []byte) (ProcessedImage, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "jpeg" && format != "png") {
		return ProcessedImage{}, ErrUnsupportedImage
	}
	if cfg.Width > maxSourcePixel || cfg.Height > maxSourcePixel {
		return ProcessedImage{}, ErrImageTooLarge
	}
	if cfg.Width < minSourceEdge || cfg.Height < minSourceEdge {
		return ProcessedImage{}, ErrImageTooSmall
	}
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrUnsupportedImage
	}
	img = downscale(flatten(img), maxOutputEdge)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return ProcessedImage{}, err
	}
	b := img.Bounds()
	return ProcessedImage{Data: buf.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

// flatten draws the image onto white so transparent PNGs do not turn black in JPEG.
func flatten(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

// downscale uses a box filter so large camera photos become small without aliasing.
func downscale(src *image.RGBA, maxEdge int) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if w <= maxEdge && h <= maxEdge {
		return src
	}
	scale := float64(maxEdge) / float64(max(w, h))
	nw, nh := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		y0, y1 := y*h/nh, max(y*h/nh+1, (y+1)*h/nh)
		for x := 0; x < nw; x++ {
			x0, x1 := x*w/nw, max(x*w/nw+1, (x+1)*w/nw)
			var r, g, b, n int
			for yy := y0; yy < y1; yy++ {
				row := src.Pix[yy*src.Stride:]
				for xx := x0; xx < x1; xx++ {
					p := row[xx*4:]
					r, g, b, n = r+int(p[0]), g+int(p[1]), b+int(p[2]), n+1
				}
			}
			o := dst.Pix[y*dst.Stride+x*4:]
			o[0], o[1], o[2], o[3] = uint8(r/n), uint8(g/n), uint8(b/n), 255
		}
	}
	return dst
}
