package photos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"

	xdraw "golang.org/x/image/draw"
)

const (
	MaxUploadBytes = 8 << 20
	minSide        = 200
	maxSide        = 10000
	maxPixels      = 40_000_000
	outputMaxSide  = 1080
	jpegQuality    = 82
)

var (
	ErrUnsupportedType = errors.New("only JPEG and PNG images are accepted")
	ErrInvalidImage    = errors.New("the file is not a valid image")
	ErrImageTooSmall   = errors.New("the image is too small (minimum 200x200 pixels)")
	ErrImageTooLarge   = errors.New("the image dimensions are too large")
)

type processedImage struct {
	Data   []byte
	Width  int
	Height int
}

// processImage validates untrusted bytes and returns a sanitised JPEG: the
// content type is sniffed (never trusted from the client), dimensions are
// checked before decoding to refuse decompression bombs, EXIF orientation is
// applied, and re-encoding drops all metadata and any trailing payload.
func processImage(raw []byte) (processedImage, error) {
	switch http.DetectContentType(raw) {
	case "image/jpeg", "image/png":
	default:
		return processedImage{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return processedImage{}, ErrInvalidImage
	}
	if cfg.Width > maxSide || cfg.Height > maxSide || cfg.Width*cfg.Height > maxPixels {
		return processedImage{}, ErrImageTooLarge
	}
	if cfg.Width < minSide || cfg.Height < minSide {
		return processedImage{}, ErrImageTooSmall
	}

	var src image.Image
	var decodeErr error
	if http.DetectContentType(raw) == "image/png" {
		src, decodeErr = png.Decode(bytes.NewReader(raw))
	} else {
		src, decodeErr = jpeg.Decode(bytes.NewReader(raw))
	}
	if decodeErr != nil {
		return processedImage{}, ErrInvalidImage
	}

	if http.DetectContentType(raw) == "image/jpeg" {
		src = applyOrientation(src, jpegOrientation(raw))
	}

	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if longest := max(w, h); longest > outputMaxSide {
		w = w * outputMaxSide / longest
		h = h * outputMaxSide / longest
	}
	w, h = max(w, 1), max(h, 1)

	// Flatten onto white: JPEG has no alpha channel.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return processedImage{}, ErrInvalidImage
	}
	return processedImage{Data: out.Bytes(), Width: w, Height: h}, nil
}

// jpegOrientation reads the EXIF orientation tag (1-8); 1 when absent.
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
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		if marker == 0xE1 {
			if o := exifOrientation(data[i+4 : i+2+size]); o != 0 {
				return o
			}
		}
		i += 2 + size
	}
	return 1
}

func exifOrientation(seg []byte) int {
	if len(seg) < 14 || string(seg[:6]) != "Exif\x00\x00" {
		return 0
	}
	t := seg[6:]
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	ifd := int(bo.Uint32(t[4:8]))
	if ifd < 8 || ifd+2 > len(t) {
		return 0
	}
	n := int(bo.Uint16(t[ifd : ifd+2]))
	for k := 0; k < n; k++ {
		e := ifd + 2 + k*12
		if e+12 > len(t) {
			return 0
		}
		if bo.Uint16(t[e:e+2]) == 0x0112 {
			v := int(bo.Uint16(t[e+8 : e+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 0
		}
	}
	return 0
}

func applyOrientation(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := o >= 5
	var dst *image.RGBA
	if swap {
		dst = image.NewRGBA(image.Rect(0, 0, h, w))
	} else {
		dst = image.NewRGBA(image.Rect(0, 0, w, h))
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
