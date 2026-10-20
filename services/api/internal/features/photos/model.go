package photos

const (
	MaxPhotosPerUser = 6
	MaxUploadBytes   = 8 << 20
	MaxDimension     = 1280
	MinDimension     = 200
	maxSourcePixels  = 40_000_000
)

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}

type StoredPhoto struct {
	ID         string
	OwnerID    string
	StorageKey string
}

type NewPhoto struct {
	StorageKey string
	Width      int
	Height     int
	SizeBytes  int
}
