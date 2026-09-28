package photos

const (
	MaxPhotos      = 6
	MaxUploadBytes = 8 << 20
	// MaxRequestBytes bounds the whole multipart request (file + envelope).
	MaxRequestBytes = MaxUploadBytes + 512<<10
)

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds"`
}
