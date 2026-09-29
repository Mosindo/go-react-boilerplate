package discovery

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

func (h *Handler) Discover(c *gin.Context) {
	profiles, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), c.Query("limit"))
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"profiles": profiles})
	case errors.Is(err, ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 20"})
	case errors.Is(err, ErrIncompleteProfile):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "complete your profile (details, preferences, location and a photo) to discover people"})
	default:
		log.Printf("discovery: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
