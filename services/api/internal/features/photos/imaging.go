package photos

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"

	xdraw "golang.org/x/image/draw"
)

const (
	maxUploadBytes = 8 << 20
	maxSourcePixel = 40_000_000 // decompression-bomb guard, checked on the header before decoding
	maxOutputEdge  = 1280
	jpegQuality    = 82
)

// ProcessImage validates an uploaded file and returns a sanitized JPEG.
// Content is sniffed (not trusted from the client), decoded and re-encoded, which strips metadata
// and any payload smuggled inside the file. Only JPEG and PNG are accepted.
func ProcessImage(raw []byte) ([]byte, error) {
	if len(raw) == 0 || len(raw) > maxUploadBytes {
		return nil, ErrInvalidImage
	}
	kind := http.DetectContentType(raw)
	if kind != "image/jpeg" && kind != "image/png" {
		return nil, ErrInvalidImage
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width < 200 || cfg.Height < 200 || cfg.Width*cfg.Height > maxSourcePixel {
		return nil, ErrInvalidImage
	}

	var src image.Image
	if kind == "image/jpeg" {
		src, err = jpeg.Decode(bytes.NewReader(raw))
	} else {
		src, err = png.Decode(bytes.NewReader(raw))
	}
	if err != nil {
		return nil, ErrInvalidImage
	}
	if kind == "image/jpeg" {
		src = applyOrientation(src, jpegOrientation(raw))
	}
	src = flattenOnWhite(src)
	src = fit(src, maxOutputEdge)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, src, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func flattenOnWhite(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

func fit(src image.Image, edge int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= edge && h <= edge {
		return src
	}
	if w >= h {
		h, w = h*edge/w, edge
	} else {
		w, h = w*edge/h, edge
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

// jpegOrientation reads the EXIF orientation tag (1-8); 1 when absent or malformed.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xDA || marker == 0xD9 {
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		if marker == 0xE1 && size >= 16 && string(data[i+4:i+10]) == "Exif\x00\x00" {
			return tiffOrientation(data[i+10 : i+2+size])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	off := int(order.Uint32(t[4:8]))
	if off < 8 || off+2 > len(t) {
		return 1
	}
	count := int(order.Uint16(t[off : off+2]))
	for n := 0; n < count; n++ {
		e := off + 2 + n*12
		if e+12 > len(t) {
			return 1
		}
		if order.Uint16(t[e:e+2]) == 0x0112 {
			if v := int(order.Uint16(t[e+8 : e+10])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// applyOrientation bakes the EXIF orientation into the pixels.
func applyOrientation(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2:
				nx, ny = w-1-x, y
			case 3:
				nx, ny = w-1-x, h-1-y
			case 4:
				nx, ny = x, h-1-y
			case 5:
				nx, ny = y, x
			case 6:
				nx, ny = h-1-y, x
			case 7:
				nx, ny = h-1-y, w-1-x
			case 8:
				nx, ny = y, w-1-x
			}
			dst.Set(nx, ny, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
