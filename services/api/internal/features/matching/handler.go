package matching

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Swipe(c *gin.Context) {
	var req swipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), c.Param("matchId")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId and action (like|pass) are required"})
	case errors.Is(err, ErrSelfSwipe):
		c.JSON(http.StatusBadRequest, gin.H{"error": "you cannot swipe yourself"})
	case errors.Is(err, ErrNotEligible), errors.Is(err, ErrMatchNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, ErrDuplicateSwipe):
		c.JSON(http.StatusConflict, gin.H{"error": "you already swiped this profile"})
	case errors.Is(err, ErrIncompleteProfile):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "complete your profile before swiping"})
	default:
		log.Printf("matching: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
