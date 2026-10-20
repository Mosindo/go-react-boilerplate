package comments

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListByPost(c *gin.Context) {
	organizationID := strings.TrimSpace(c.GetString("organizationID"))
	if organizationID == "" {
		logger.LogHandlerError(c, "comments.list.organization", http.StatusUnauthorized, errors.New("missing organization"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	postID := strings.TrimSpace(c.Param("postId"))
	if !isUUIDLike(postID) {
		logger.LogHandlerError(c, "comments.list.validate_post_id", http.StatusBadRequest, errors.New("invalid post id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	limit := parseBoundedLimit(c.Query("limit"), 20, 200)
	offset := parseNonNegativeInt(c.Query("offset"), 0)
	comments, err := h.service.ListByPost(c.Request.Context(), organizationID, postID, limit, offset)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrPostNotFound):
			status = http.StatusNotFound
			c.JSON(status, gin.H{"error": "post not found"})
		default:
			c.JSON(status, gin.H{"error": "could not fetch comments"})
		}
		logger.LogHandlerError(c, "comments.list", status, err)
		return
	}

	payload := make([]CommentResponse, 0, len(comments))
	for _, comment := range comments {
		payload = append(payload, mapCommentResponse(comment))
	}

	c.JSON(http.StatusOK, CommentsResponse{Comments: payload})
}

func (h *Handler) Create(c *gin.Context) {
	userID := c.GetString("userID")
	organizationID := strings.TrimSpace(c.GetString("organizationID"))
	if organizationID == "" {
		logger.LogHandlerError(c, "comments.create.organization", http.StatusUnauthorized, errors.New("missing organization"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	postID := strings.TrimSpace(c.Param("postId"))
	if !isUUIDLike(postID) {
		logger.LogHandlerError(c, "comments.create.validate_post_id", http.StatusBadRequest, errors.New("invalid post id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "comments.create.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	comment, err := h.service.Create(c.Request.Context(), organizationID, postID, userID, req.Content)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrPostNotFound):
			status = http.StatusNotFound
			c.JSON(status, gin.H{"error": "post not found"})
		case errors.Is(err, ErrCommentContentNeeded):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "comment content required"})
		case errors.Is(err, ErrCommentContentTooLong):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "comment content too long"})
		default:
			c.JSON(status, gin.H{"error": "could not create comment"})
		}
		logger.LogHandlerError(c, "comments.create", status, err)
		return
	}

	status := http.StatusCreated
	c.JSON(status, mapCommentResponse(comment))
	logger.LogHandlerEvent(c, "comments.create.success", status, map[string]string{
		"post_id":    comment.PostID,
		"comment_id": comment.ID,
	})
}

func mapCommentResponse(comment Comment) CommentResponse {
	return CommentResponse{
		ID:           comment.ID,
		PostID:       comment.PostID,
		AuthorUserID: comment.AuthorUserID,
		Content:      comment.Content,
		CreatedAt:    comment.CreatedAt,
		UpdatedAt:    comment.UpdatedAt,
	}
}

func parsePositiveInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseBoundedLimit(raw string, fallback, max int) int {
	parsed := parsePositiveInt(raw, fallback)
	if parsed > max {
		return max
	}
	return parsed
}

func parseNonNegativeInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func isUUIDLike(v string) bool {
	if len(v) != 36 {
		return false
	}
	hyphenPos := map[int]struct{}{8: {}, 13: {}, 18: {}, 23: {}}
	for i := 0; i < len(v); i++ {
		ch := v[i]
		if _, ok := hyphenPos[i]; ok {
			if ch != '-' {
				return false
			}
			continue
		}
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}
