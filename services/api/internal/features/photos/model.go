package photos

const (
	MaxPhotos       = 6
	MaxUploadBytes  = 5 << 20 // raw upload limit
	MaxPixels       = 16_000_000
	MaxDimension    = 1280 // longest side after processing
	JPEGQuality     = 82
	maxConcurrentIO = 4
)

type PhotoResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type PhotosResponse struct {
	Photos []PhotoResponse `json:"photos"`
}

type ReorderRequest struct {
	IDs []string `json:"ids" binding:"required,max=6"`
}
