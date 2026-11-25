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
)

const (
	maxSourcePixels = 40_000_000
	// maxDecodeBytes bounds the memory needed to decode one image (decoded
	// buffer + RGBA working copy), whatever the compressed file size is.
	maxDecodeBytes = 192 << 20
	maxOutputSide  = 1440
	minSourceSide  = 200
	jpegQuality    = 82
)

var (
	errUnsupportedImage = errors.New("unsupported image")
	errImageTooSmall    = errors.New("image too small")
	errImageTooLarge    = errors.New("image dimensions too large")
)

// processImage validates an uploaded image and re-encodes it as a clean JPEG.
// Re-encoding drops every metadata block (EXIF, GPS, XMP...) and neutralises
// polyglot files; EXIF orientation is applied first so photos stay upright.
func processImage(data []byte) ([]byte, int, int, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return nil, 0, 0, errUnsupportedImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxSourcePixels {
		return nil, 0, 0, errImageTooLarge
	}
	if int64(cfg.Width)*int64(cfg.Height)*(bytesPerPixel(cfg.ColorModel)+4) > maxDecodeBytes {
		return nil, 0, 0, errImageTooLarge
	}

	// Only a couple of images are decoded at the same time to cap memory.
	decodeSlots <- struct{}{}
	defer func() { <-decodeSlots }()
	if cfg.Width < minSourceSide || cfg.Height < minSourceSide {
		return nil, 0, 0, errImageTooSmall
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, errUnsupportedImage
	}

	rgba := image.NewRGBA(image.Rect(0, 0, src.Bounds().Dx(), src.Bounds().Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, src.Bounds().Min, draw.Src)

	if format == "jpeg" {
		rgba = applyOrientation(rgba, jpegOrientation(data))
	}
	out := downscale(rgba, maxOutputSide)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), out.Bounds().Dx(), out.Bounds().Dy(), nil
}

var decodeSlots = make(chan struct{}, 2)

// bytesPerPixel estimates the decoded size of one pixel for a colour model.
func bytesPerPixel(m color.Model) int64 {
	switch m {
	case color.RGBA64Model, color.NRGBA64Model:
		return 8
	case color.Gray16Model:
		return 2
	case color.GrayModel:
		return 1
	default:
		return 4
	}
}

// downscale reduces img so its longest side is at most maxSide, using an
// area-average (box) filter which gives good quality for reductions.
func downscale(img *image.RGBA, maxSide int) *image.RGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	longest := w
	if h > longest {
		longest = h
	}
	if longest <= maxSide {
		return img
	}
	nw := w * maxSide / longest
	nh := h * maxSide / longest
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy0 := y * h / nh
		sy1 := (y + 1) * h / nh
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < nw; x++ {
			sx0 := x * w / nw
			sx1 := (x + 1) * w / nw
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, b, a, n uint32
			for sy := sy0; sy < sy1; sy++ {
				row := img.Pix[sy*img.Stride:]
				for sx := sx0; sx < sx1; sx++ {
					p := row[sx*4 : sx*4+4]
					r += uint32(p[0])
					g += uint32(p[1])
					b += uint32(p[2])
					a += uint32(p[3])
					n++
				}
			}
			d := dst.Pix[y*dst.Stride+x*4:]
			d[0], d[1], d[2], d[3] = uint8(r/n), uint8(g/n), uint8(b/n), uint8(a/n)
		}
	}
	return dst
}

// applyOrientation rotates/flips img according to an EXIF orientation value.
func applyOrientation(img *image.RGBA, orientation int) *image.RGBA {
	if orientation < 2 || orientation > 8 {
		return img
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	dw, dh := w, h
	if orientation >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch orientation {
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
			copy(dst.Pix[dy*dst.Stride+dx*4:dy*dst.Stride+dx*4+4], img.Pix[y*img.Stride+x*4:y*img.Stride+x*4+4])
		}
	}
	return dst
}

// jpegOrientation extracts the EXIF orientation tag (0x0112) from a JPEG.
// It returns 1 (normal) when absent or unreadable.
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
		segment := data[i+4 : i+2+size]
		if marker == 0xE1 && len(segment) > 14 && string(segment[:6]) == "Exif\x00\x00" {
			return tiffOrientation(segment[6:])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(tiff []byte) int {
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
	offset := int(order.Uint32(tiff[4:8]))
	if offset < 8 || offset+2 > len(tiff) {
		return 1
	}
	count := int(order.Uint16(tiff[offset : offset+2]))
	for e := 0; e < count; e++ {
		entry := offset + 2 + e*12
		if entry+12 > len(tiff) {
			return 1
		}
		if order.Uint16(tiff[entry:entry+2]) == 0x0112 {
			v := int(order.Uint16(tiff[entry+8 : entry+10]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}
