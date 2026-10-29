package photos

const (
	MaxPhotosPerUser = 6
	MaxUploadBytes   = 8 << 20
	maxDimension     = 1600
	maxSourceSide    = 12000
	maxSourcePixels  = 48_000_000
	jpegQuality      = 85
)

type Photo struct {
	ID          string
	UserID      string
	StorageKey  string
	ContentType string
	SizeBytes   int
	Width       int
	Height      int
	Position    int
}

type PhotoResponse struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,max=20"`
}
