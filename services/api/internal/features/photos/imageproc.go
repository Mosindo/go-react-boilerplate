package photos

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png" // register the PNG decoder
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
)

var (
	errUnsupportedType = &platformerrors.AppError{Status: http.StatusUnsupportedMediaType, Message: "only JPEG and PNG images are accepted"}
	errTooLarge        = &platformerrors.AppError{Status: http.StatusRequestEntityTooLarge, Message: "image is too large (max 8 MB)"}
	errBadImage        = platformerrors.BadRequest("image is invalid, too small (min 200 px) or has excessive dimensions")
)

// ProcessImage validates untrusted bytes and returns a sanitized JPEG: the
// type is sniffed from content (never from the client's header), the pixel
// count is bounded before decoding, EXIF orientation is applied, metadata
// (EXIF/GPS) is dropped by re-encoding and the longest side is capped.
func ProcessImage(data []byte) (out []byte, width, height int, err error) {
	if len(data) > MaxUploadBytes {
		return nil, 0, 0, errTooLarge
	}
	head := data
	if len(head) > 512 {
		head = head[:512]
	}
	contentType := http.DetectContentType(head)
	if contentType != "image/jpeg" && contentType != "image/png" {
		return nil, 0, 0, errUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxSourcePixels {
		return nil, 0, 0, errBadImage
	}
	if cfg.Width < MinDimension || cfg.Height < MinDimension {
		return nil, 0, 0, errBadImage
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, errBadImage
	}

	if contentType == "image/jpeg" {
		src = applyOrientation(src, jpegOrientation(data))
	}
	flat := fitWithin(flatten(src), MaxDimension)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, flat, &jpeg.Options{Quality: 85}); err != nil {
		return nil, 0, 0, err
	}
	b := flat.Bounds()
	return buf.Bytes(), b.Dx(), b.Dy(), nil
}

// flatten composites transparency over white and returns an opaque RGBA copy.
func flatten(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

// fitWithin downsizes with an area average (box) filter; images already
// within the limit are returned untouched.
func fitWithin(src *image.RGBA, maxSide int) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if w <= maxSide && h <= maxSide {
		return src
	}
	scale := float64(maxSide) / float64(w)
	if h > w {
		scale = float64(maxSide) / float64(h)
	}
	nw, nh := int(float64(w)*scale), int(float64(h)*scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		y0, y1 := y*h/nh, (y+1)*h/nh
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < nw; x++ {
			x0, x1 := x*w/nw, (x+1)*w/nw
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, n uint32
			for yy := y0; yy < y1; yy++ {
				row := src.Pix[yy*src.Stride:]
				for xx := x0; xx < x1; xx++ {
					p := row[xx*4:]
					r += uint32(p[0])
					g += uint32(p[1])
					b += uint32(p[2])
					n++
				}
			}
			o := dst.Pix[y*dst.Stride+x*4:]
			o[0], o[1], o[2], o[3] = uint8(r/n), uint8(g/n), uint8(b/n), 255
		}
	}
	return dst
}

// jpegOrientation reads the EXIF orientation tag (1-8), defaulting to 1.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			return 1
		}
		marker := data[pos+1]
		if marker == 0xDA || marker == 0xD9 { // start of scan / end: no more headers
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if size < 2 || pos+2+size > len(data) {
			return 1
		}
		segment := data[pos+4 : pos+2+size]
		if marker == 0xE1 && len(segment) >= 14 && string(segment[:6]) == "Exif\x00\x00" {
			return exifOrientation(segment[6:])
		}
		pos += 2 + size
	}
	return 1
}

func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	ifd := int(order.Uint32(tiff[4:8]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 1
	}
	count := int(order.Uint16(tiff[ifd : ifd+2]))
	for i := 0; i < count; i++ {
		entry := ifd + 2 + i*12
		if entry+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[entry:entry+2]) == 0x0112 {
			value := int(order.Uint16(tiff[entry+8 : entry+10]))
			if value >= 1 && value <= 8 {
				return value
			}
			return 1
		}
	}
	return 1
}

// applyOrientation rotates/flips so the pixels match what the camera showed.
func applyOrientation(src image.Image, orientation int) image.Image {
	if orientation <= 1 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := orientation >= 5
	dw, dh := w, h
	if swap {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch orientation {
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
