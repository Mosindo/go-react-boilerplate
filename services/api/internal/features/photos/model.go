package photos

const (
	MaxPhotos       = 6
	MaxUploadBytes  = 6 << 20
	MaxSourcePixels = 25_000_000
	MaxSourceSide   = 8000
	MinSide         = 200
	FullMaxSide     = 1280
	ThumbMaxSide    = 360
)

type Processed struct {
	Width     int
	Height    int
	Data      []byte
	Thumbnail []byte
}

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type StoredImage struct {
	OwnerID     string
	ContentType string
	Data        []byte
}

type ReorderRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=6"`
}
