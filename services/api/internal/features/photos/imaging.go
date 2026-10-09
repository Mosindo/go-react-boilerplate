package photos

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder
	"io"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder
)

var (
	ErrUnsupportedImage = errors.New("photos: unsupported or corrupt image")
	ErrImageTooSmall    = errors.New("photos: image too small")
	ErrImageTooLarge    = errors.New("photos: image dimensions too large")
)

// ProcessedImage is a sanitised JPEG ready to store.
type ProcessedImage struct {
	Data          []byte
	Width, Height int
}

// Process validates untrusted image bytes and re-encodes them. Re-encoding is the
// main upload defence: the output is a fresh JPEG built from decoded pixels, so
// polyglot payloads, EXIF/GPS metadata and trailing data never reach storage.
func Process(raw []byte) (ProcessedImage, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrUnsupportedImage
	}
	if format != "jpeg" && format != "png" && format != "webp" {
		return ProcessedImage{}, ErrUnsupportedImage
	}
	if cfg.Width < minSide || cfg.Height < minSide {
		return ProcessedImage{}, ErrImageTooSmall
	}
	// Check the declared size *before* decoding to stop decompression bombs.
	if cfg.Width > 20000 || cfg.Height > 20000 || cfg.Width*cfg.Height > maxSourcePixels {
		return ProcessedImage{}, ErrImageTooLarge
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return ProcessedImage{}, ErrUnsupportedImage
	}
	if format == "jpeg" {
		img = applyOrientation(img, jpegOrientation(raw))
	}
	img = fit(img, maxSide)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, flatten(img), &jpeg.Options{Quality: jpegQuality}); err != nil {
		return ProcessedImage{}, err
	}
	b := img.Bounds()
	return ProcessedImage{Data: out.Bytes(), Width: b.Dx(), Height: b.Dy()}, nil
}

// flatten draws onto an opaque white canvas because JPEG has no alpha channel.
func flatten(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	xdraw.Draw(dst, dst.Bounds(), image.White, image.Point{}, xdraw.Src)
	xdraw.Draw(dst, dst.Bounds(), src, b.Min, xdraw.Over)
	return dst
}

// fit downsizes so that the longest side is at most limit pixels.
func fit(src image.Image, limit int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= limit && h <= limit {
		return src
	}
	var nw, nh int
	if w >= h {
		nw, nh = limit, h*limit/w
	} else {
		nw, nh = w*limit/h, limit
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

// jpegOrientation reads the EXIF orientation tag (1-8); 1 when absent or malformed.
func jpegOrientation(data []byte) int {
	r := bytes.NewReader(data)
	var soi [2]byte
	if _, err := io.ReadFull(r, soi[:]); err != nil || soi != [2]byte{0xFF, 0xD8} {
		return 1
	}
	for {
		var marker [2]byte
		if _, err := io.ReadFull(r, marker[:]); err != nil || marker[0] != 0xFF {
			return 1
		}
		if marker[1] == 0xDA || marker[1] == 0xD9 { // start of scan / end of image
			return 1
		}
		var lenBuf [2]byte
		if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
			return 1
		}
		segLen := int(binary.BigEndian.Uint16(lenBuf[:])) - 2
		if segLen < 0 {
			return 1
		}
		seg := make([]byte, segLen)
		if _, err := io.ReadFull(r, seg); err != nil {
			return 1
		}
		if marker[1] == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			return parseExifOrientation(seg[6:])
		}
	}
}

func parseExifOrientation(tiff []byte) int {
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
		off := ifd + 2 + i*12
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

// applyOrientation rotates/flips pixels so the image displays upright without EXIF.
func applyOrientation(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	swap := o >= 5
	dw, dh := w, h
	if swap {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2: // mirror horizontal
				nx, ny = w-1-x, y
			case 3: // rotate 180
				nx, ny = w-1-x, h-1-y
			case 4: // mirror vertical
				nx, ny = x, h-1-y
			case 5: // transpose
				nx, ny = y, x
			case 6: // rotate 90 CW
				nx, ny = h-1-y, x
			case 7: // transverse
				nx, ny = h-1-y, w-1-x
			case 8: // rotate 90 CCW
				nx, ny = y, w-1-x
			}
			dst.Set(nx, ny, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
