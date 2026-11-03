package photos

const (
	MaxPhotos     = 6
	MaxUploadSize = 8 << 20 // 8 MiB of raw upload
	MaxDimension  = 1280    // longest side after re-encoding
	maxPixels     = 50_000_000
)

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type OrderRequest struct {
	IDs []string `json:"ids"`
}
