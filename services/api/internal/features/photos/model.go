package photos

const MaxPhotos = 6

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}

type storedPhoto struct {
	ID         string
	StorageKey string
	Width      int
	Height     int
	Position   int
}
