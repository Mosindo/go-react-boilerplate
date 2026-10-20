package posts

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

func (h *Handler) List(c *gin.Context) {
	organizationID := strings.TrimSpace(c.GetString("organizationID"))
	if organizationID == "" {
		logger.LogHandlerError(c, "posts.list.organization", http.StatusUnauthorized, errors.New("missing organization"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	limit := parseBoundedLimit(c.Query("limit"), 20, 100)
	offset := parseNonNegativeInt(c.Query("offset"), 0)
	posts, err := h.service.ListWithPagination(c.Request.Context(), organizationID, limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "posts.list", http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch posts"})
		return
	}

	payload := make([]PostResponse, 0, len(posts))
	for _, post := range posts {
		payload = append(payload, mapPostResponse(post))
	}

	c.JSON(http.StatusOK, PostsResponse{Posts: payload})
}

func (h *Handler) Create(c *gin.Context) {
	userID := c.GetString("userID")
	organizationID := strings.TrimSpace(c.GetString("organizationID"))
	if organizationID == "" {
		logger.LogHandlerError(c, "posts.create.organization", http.StatusUnauthorized, errors.New("missing organization"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.LogHandlerError(c, "posts.create.bind", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	post, err := h.service.Create(c.Request.Context(), organizationID, userID, req.Title, req.Body)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrTitleRequired):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "post title required"})
		case errors.Is(err, ErrBodyRequired):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "post body required"})
		case errors.Is(err, ErrTitleTooLong):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "post title too long"})
		case errors.Is(err, ErrBodyTooLong):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "post body too long"})
		default:
			c.JSON(status, gin.H{"error": "could not create post"})
		}
		logger.LogHandlerError(c, "posts.create", status, err)
		return
	}

	status := http.StatusCreated
	c.JSON(status, mapPostResponse(post))
	logger.LogHandlerEvent(c, "posts.create.success", status, map[string]string{"post_id": post.ID})
}

func (h *Handler) GetByID(c *gin.Context) {
	organizationID := strings.TrimSpace(c.GetString("organizationID"))
	if organizationID == "" {
		logger.LogHandlerError(c, "posts.get.organization", http.StatusUnauthorized, errors.New("missing organization"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	postID := strings.TrimSpace(c.Param("postId"))
	if !isUUIDLike(postID) {
		logger.LogHandlerError(c, "posts.get.validate_post_id", http.StatusBadRequest, errors.New("invalid post id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	post, err := h.service.GetByID(c.Request.Context(), organizationID, postID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrPostNotFound):
			status = http.StatusNotFound
			c.JSON(status, gin.H{"error": "post not found"})
		default:
			c.JSON(status, gin.H{"error": "could not fetch post"})
		}
		logger.LogHandlerError(c, "posts.get", status, err)
		return
	}

	c.JSON(http.StatusOK, mapPostResponse(post))
}

func mapPostResponse(post Post) PostResponse {
	return PostResponse{
		ID:           post.ID,
		AuthorUserID: post.AuthorUserID,
		Title:        post.Title,
		Body:         post.Body,
		CreatedAt:    post.CreatedAt,
		UpdatedAt:    post.UpdatedAt,
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
