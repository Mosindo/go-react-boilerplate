package photos

type Photo struct {
	ID       string `json:"id"`
	Position int16  `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	URL      string `json:"url"`
}

type OrderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}

type stored struct {
	Photo
	UserID     string
	StorageKey string
}
