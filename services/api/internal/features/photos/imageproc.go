package photos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register PNG decoder

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // register WebP decoder
)

var (
	ErrUnsupportedImage = errors.New("unsupported or corrupt image")
	ErrImageTooLarge    = errors.New("image dimensions too large")
)

const (
	fullMaxEdge   = 1080
	thumbMaxEdge  = 320
	jpegQuality   = 82
	maxEdgePixels = 10000
	maxPixels     = 24_000_000
)

type processed struct {
	Full, Thumb   []byte
	Width, Height int
}

// processImage validates untrusted bytes and re-encodes them as clean JPEGs:
//   - the real format is detected from content (not from client headers or file names),
//   - dimensions are checked before decoding to defeat decompression bombs,
//   - EXIF orientation is applied, then all metadata (GPS included) is dropped by re-encoding.
func processImage(data []byte) (processed, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return processed{}, ErrUnsupportedImage
	}
	if format != "jpeg" && format != "png" && format != "webp" {
		return processed{}, ErrUnsupportedImage
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxEdgePixels || cfg.Height > maxEdgePixels || cfg.Width*cfg.Height > maxPixels {
		return processed{}, ErrImageTooLarge
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return processed{}, ErrUnsupportedImage
	}
	if format == "jpeg" {
		img = applyOrientation(img, exifOrientation(data))
	}
	rgba := flatten(img)

	full, err := encodeScaled(rgba, fullMaxEdge)
	if err != nil {
		return processed{}, err
	}
	thumb, err := encodeScaled(rgba, thumbMaxEdge)
	if err != nil {
		return processed{}, err
	}
	b := rgba.Bounds()
	w, h := scaledSize(b.Dx(), b.Dy(), fullMaxEdge)
	return processed{Full: full, Thumb: thumb, Width: w, Height: h}, nil
}

// flatten draws transparent images over white so JPEG encoding is predictable.
func flatten(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

func scaledSize(w, h, maxEdge int) (int, int) {
	if w <= maxEdge && h <= maxEdge {
		return w, h
	}
	if w >= h {
		nh := h * maxEdge / w
		if nh < 1 {
			nh = 1
		}
		return maxEdge, nh
	}
	nw := w * maxEdge / h
	if nw < 1 {
		nw = 1
	}
	return nw, maxEdge
}

func encodeScaled(src *image.RGBA, maxEdge int) ([]byte, error) {
	w, h := scaledSize(src.Bounds().Dx(), src.Bounds().Dy(), maxEdge)
	var out image.Image = src
	if w != src.Bounds().Dx() || h != src.Bounds().Dy() {
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
		out = dst
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// exifOrientation extracts the EXIF orientation (1-8) from a JPEG, defaulting to 1.
func exifOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xD9 || marker == 0xDA { // EOI / start of scan: no more metadata
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		seg := data[i+4 : i+2+size]
		if marker == 0xE1 && len(seg) >= 14 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
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
	ifd := int(order.Uint32(t[4:8]))
	if ifd < 8 || ifd+2 > len(t) {
		return 1
	}
	n := int(order.Uint16(t[ifd : ifd+2]))
	for k := 0; k < n; k++ {
		off := ifd + 2 + k*12
		if off+12 > len(t) {
			return 1
		}
		if order.Uint16(t[off:off+2]) == 0x0112 {
			if v := int(order.Uint16(t[off+8 : off+10])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

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
