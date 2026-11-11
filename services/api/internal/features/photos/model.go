package photos

type StoredPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6,dive,uuid"`
}
