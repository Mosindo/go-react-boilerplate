package photos

const (
	MaxPhotos     = 6
	MaxUploadSize = 8 << 20 // raw upload limit
	maxPixels     = 40_000_000
	maxSide       = 1280
	jpegQuality   = 82
)

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

// Content is a stored, already-sanitized JPEG.
type Content struct {
	ContentType string
	Data        []byte
}

type OrderRequest struct {
	IDs []string `json:"ids"`
}
