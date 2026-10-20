package comments

import "time"

type Comment struct {
	ID           string
	PostID       string
	AuthorUserID string
	Content      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CommentResponse struct {
	ID           string    `json:"id"`
	PostID       string    `json:"postId"`
	AuthorUserID string    `json:"authorUserId"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CommentsResponse struct {
	Comments []CommentResponse `json:"comments"`
}

type CreateCommentRequest struct {
	Content string `json:"content" binding:"required,max=4000"`
}
