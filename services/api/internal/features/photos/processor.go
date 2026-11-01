package photos

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png" // decodeur PNG
	"net/http"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // decodeur WebP
)

// Process valide une image uploadee puis la re-encode en JPEG redimensionne.
// Le re-encodage supprime les metadonnees (EXIF/GPS) et neutralise les fichiers
// polyglottes: seul un pixmap decode puis recompresse est conserve.
func Process(data []byte) (out []byte, width, height int, err error) {
	if len(data) == 0 {
		return nil, 0, 0, ErrInvalidImage
	}
	if len(data) > MaxUploadBytes {
		return nil, 0, 0, ErrTooLarge
	}
	switch http.DetectContentType(data) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return nil, 0, 0, ErrInvalidImage
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 {
		return nil, 0, 0, ErrInvalidImage
	}
	if cfg.Width < 200 || cfg.Height < 200 {
		return nil, 0, 0, ErrInvalidImage
	}
	if int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return nil, 0, 0, ErrTooLarge
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, ErrInvalidImage
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > MaxSide || h > MaxSide {
		if w >= h {
			h = h * MaxSide / w
			w = MaxSide
		} else {
			w = w * MaxSide / h
			h = MaxSide
		}
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
	}
	// Fond blanc: aplatit la transparence (PNG/WebP) avant l'encodage JPEG.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), w, h, nil
}
