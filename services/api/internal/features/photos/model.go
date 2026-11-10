package photos

const (
	MaxPhotos       = 6
	MaxUploadBytes  = 10 << 20 // raw upload limit
	MaxSourcePixels = 40_000_000
	MinSourceSide   = 200
	MaxOutputSide   = 1440
	JPEGQuality     = 85
)

type Photo struct {
	ID         string
	Position   int
	StorageKey string
	Width      int
	Height     int
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6,dive,uuid"`
}
