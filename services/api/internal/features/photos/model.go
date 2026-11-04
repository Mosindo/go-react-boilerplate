package photos

// MaxPhotos is the maximum number of photos per profile.
const MaxPhotos = 6

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type OrderRequest struct {
	IDs []string `json:"ids" binding:"required,max=9"`
}

type storedPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
}
