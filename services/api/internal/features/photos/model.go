package photos

const MaxPhotos = 6

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type PhotosResponse struct {
	Photos []Photo `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,max=6"`
}

type storedPhoto struct {
	ID         string
	StorageKey string
	Position   int
	Width      int
	Height     int
}
