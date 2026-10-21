package photos

const MaxPhotosPerUser = 6

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
	Position int    `json:"position"`
}

type OrderRequest struct {
	PhotoIDs []string `json:"photoIds"`
}

type storedPhoto struct {
	ID       string
	Position int
	Key      string
	ThumbKey string
}
