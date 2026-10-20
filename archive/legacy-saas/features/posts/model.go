package posts

import "time"

type Post struct {
	ID           string
	AuthorUserID string
	Title        string
	Body         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PostResponse struct {
	ID           string    `json:"id"`
	AuthorUserID string    `json:"authorUserId"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type PostsResponse struct {
	Posts []PostResponse `json:"posts"`
}

type CreatePostRequest struct {
	Title string `json:"title" binding:"required,max=160"`
	Body  string `json:"body" binding:"required,max=10000"`
}
