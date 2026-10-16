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
	"io"
)

const (
	MaxUploadBytes = 8 << 20
	maxSourceSide  = 8000
	maxSourcePixel = 12_000_000
	minSide        = 200
	outputMaxSide  = 1280
	jpegQuality    = 82
)

var (
	ErrUnsupportedType = errors.New("only JPEG and PNG images are accepted")
	ErrTooLarge        = errors.New("image file is too large")
	ErrBadDimensions   = errors.New("image dimensions are not acceptable")
	ErrCorruptImage    = errors.New("image could not be read")
)

type Processed struct {
	Data   []byte
	Width  int
	Height int
}

// Process validates untrusted upload bytes and returns a clean, re-encoded JPEG.
// The type is decided from the content (never the client-supplied name or MIME),
// decoding is bounded against decompression bombs, EXIF orientation is applied, and
// re-encoding drops all metadata (GPS, device info) and any appended payload.
func Process(r io.Reader) (Processed, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return Processed{}, ErrCorruptImage
	}
	if len(raw) > MaxUploadBytes {
		return Processed{}, ErrTooLarge
	}

	isJPEG := bytes.HasPrefix(raw, []byte{0xFF, 0xD8, 0xFF})
	isPNG := bytes.HasPrefix(raw, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1A, '\n'})
	if !isJPEG && !isPNG {
		return Processed{}, ErrUnsupportedType
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return Processed{}, ErrCorruptImage
	}
	if cfg.Width < minSide || cfg.Height < minSide ||
		cfg.Width > maxSourceSide || cfg.Height > maxSourceSide ||
		cfg.Width*cfg.Height > maxSourcePixel {
		return Processed{}, ErrBadDimensions
	}

	var src image.Image
	if isJPEG {
		src, err = jpeg.Decode(bytes.NewReader(raw))
	} else {
		src, err = png.Decode(bytes.NewReader(raw))
	}
	if err != nil {
		return Processed{}, ErrCorruptImage
	}

	// Flatten onto white: JPEG has no alpha, and this normalises every colour model.
	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Over)

	if isJPEG {
		rgba = applyOrientation(rgba, exifOrientation(raw))
	}
	rgba = fit(rgba, outputMaxSide)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, rgba, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return Processed{}, ErrCorruptImage
	}
	return Processed{Data: out.Bytes(), Width: rgba.Bounds().Dx(), Height: rgba.Bounds().Dy()}, nil
}

// exifOrientation extracts the EXIF orientation tag (1-8) from a JPEG, or 1.
func exifOrientation(data []byte) int {
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
		if marker == 0xE1 && size >= 16 && bytes.Equal(data[i+4:i+10], []byte("Exif\x00\x00")) {
			return parseTIFFOrientation(data[i+10 : i+2+size])
		}
		i += 2 + size
	}
	return 1
}

func parseTIFFOrientation(tiff []byte) int {
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
	for n := 0; n < count; n++ {
		off := ifd + 2 + n*12
		if off+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[off:off+2]) == 0x0112 {
			v := int(order.Uint16(tiff[off+8 : off+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// applyOrientation maps each destination pixel back to its source according to the
// EXIF orientation value.
func applyOrientation(src *image.RGBA, o int) *image.RGBA {
	if o <= 1 || o > 8 {
		return src
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var sx, sy int
			switch o {
			case 2:
				sx, sy = w-1-x, y
			case 3:
				sx, sy = w-1-x, h-1-y
			case 4:
				sx, sy = x, h-1-y
			case 5:
				sx, sy = y, x
			case 6:
				sx, sy = y, h-1-x
			case 7:
				sx, sy = w-1-y, h-1-x
			case 8:
				sx, sy = w-1-y, x
			}
			si := src.PixOffset(sx, sy)
			di := dst.PixOffset(x, y)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}

// fit downsizes (never upsizes) so the longest side is at most max, averaging the
// source pixels covered by each destination pixel.
func fit(src *image.RGBA, max int) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	if w <= max && h <= max {
		return src
	}
	dw, dh := max, max
	if w >= h {
		dh = h * max / w
	} else {
		dw = w * max / h
	}
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		y0, y1 := y*h/dh, (y+1)*h/dh
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < dw; x++ {
			x0, x1 := x*w/dw, (x+1)*w/dw
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, n int
			for sy := y0; sy < y1; sy++ {
				row := src.PixOffset(x0, sy)
				for sx := x0; sx < x1; sx++ {
					r += int(src.Pix[row])
					g += int(src.Pix[row+1])
					b += int(src.Pix[row+2])
					row += 4
					n++
				}
			}
			di := dst.PixOffset(x, y)
			dst.Pix[di], dst.Pix[di+1], dst.Pix[di+2], dst.Pix[di+3] = uint8(r/n), uint8(g/n), uint8(b/n), 255
		}
	}
	return dst
}
