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

type processed struct {
	Full  []byte
	Thumb []byte
}

// process validates an uploaded image and re-encodes it. Re-encoding drops metadata (EXIF, GPS) and any
// payload appended to the file, so a polyglot upload cannot be served back as something else.
func process(data []byte) (processed, error) {
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	if ct := http.DetectContentType(sniff); ct != "image/jpeg" && ct != "image/png" {
		return processed{}, ErrInvalidImage
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return processed{}, ErrInvalidImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return processed{}, ErrInvalidImage
	}
	// Reject decompression bombs before allocating pixels.
	if cfg.Width > maxSourceSide || cfg.Height > maxSourceSide || cfg.Width*cfg.Height > maxSourcePixels {
		return processed{}, ErrImageTooLarge
	}

	var img image.Image
	switch format {
	case "jpeg":
		img, err = jpeg.Decode(bytes.NewReader(data))
	default:
		img, err = png.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return processed{}, ErrInvalidImage
	}
	if format == "jpeg" {
		img = applyOrientation(img, jpegOrientation(data))
	}

	full, err := encodeJPEG(fit(img, fullMaxSide))
	if err != nil {
		return processed{}, err
	}
	thumb, err := encodeJPEG(fit(img, thumbMaxSide))
	if err != nil {
		return processed{}, err
	}
	return processed{Full: full, Thumb: thumb}, nil
}

// fit scales img down so that its longest side is at most maxSide, and flattens transparency on white.
func fit(img image.Image, maxSide int) *image.RGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > maxSide || h > maxSide {
		if w >= h {
			h = h * maxSide / w
			w = maxSide
		} else {
			w = w * maxSide / h
			h = maxSide
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// jpegOrientation reads the EXIF orientation tag (1..8); 1 when absent or malformed.
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
		if marker == 0xDA || marker == 0xD9 {
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if size < 2 || pos+2+size > len(data) {
			return 1
		}
		if marker == 0xE1 {
			if o := exifOrientation(data[pos+4 : pos+2+size]); o != 0 {
				return o
			}
		}
		pos += 2 + size
	}
	return 1
}

func exifOrientation(seg []byte) int {
	if len(seg) < 14 || string(seg[:6]) != "Exif\x00\x00" {
		return 0
	}
	tiff := seg[6:]
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0
	}
	ifd := int(order.Uint32(tiff[4:8]))
	if ifd < 8 || ifd+2 > len(tiff) {
		return 0
	}
	count := int(order.Uint16(tiff[ifd : ifd+2]))
	for i := 0; i < count; i++ {
		entry := ifd + 2 + i*12
		if entry+12 > len(tiff) {
			return 0
		}
		if order.Uint16(tiff[entry:entry+2]) == 0x0112 {
			v := int(order.Uint16(tiff[entry+8 : entry+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 0
		}
	}
	return 0
}

// applyOrientation returns img transformed so that it displays upright.
func applyOrientation(img image.Image, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return img
	}
	b := img.Bounds()
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
			dst.Set(nx, ny, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
