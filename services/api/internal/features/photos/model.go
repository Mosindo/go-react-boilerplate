package photos

type OrderRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=6,dive,max=64"`
}

type PhotoResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type PhotosResponse struct {
	Photos []PhotoResponse `json:"photos"`
}
