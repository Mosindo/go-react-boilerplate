package photos

const MaxPhotosPerUser = 6

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds"`
}

type storedPhoto struct {
	ID       string
	Position int
}
