package photos

const MaxPhotos = 6

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type StoredPhoto struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	Width      int
	Height     int
}

type PhotosResponse struct {
	Photos []Photo `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}
