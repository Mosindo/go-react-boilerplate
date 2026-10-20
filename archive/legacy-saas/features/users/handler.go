package users

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
		logger.LogHandlerError(c, "users.list.organization", http.StatusUnauthorized, errors.New("missing organization"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing organization"})
		return
	}

	limit := parseBoundedLimit(c.Query("limit"), 20, 100)
	offset := parseNonNegativeInt(c.Query("offset"), 0)
	users, err := h.service.ListWithPagination(c.Request.Context(), organizationID, limit, offset)
	if err != nil {
		logger.LogHandlerError(c, "users.list", http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch users"})
		return
	}

	payload := make([]UserResponse, 0, len(users))
	for _, user := range users {
		payload = append(payload, UserResponse{
			ID:             user.ID,
			Email:          user.Email,
			OrganizationID: user.OrganizationID,
			CreatedAt:      user.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, UsersResponse{Users: payload})
}

func (h *Handler) GetByID(c *gin.Context) {
	organizationID := strings.TrimSpace(c.GetString("organizationID"))
	if organizationID == "" {
		logger.LogHandlerError(c, "users.get.organization", http.StatusUnauthorized, errors.New("missing organization"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing organization"})
		return
	}

	userID := strings.TrimSpace(c.Param("userId"))
	if !isUUIDLike(userID) {
		logger.LogHandlerError(c, "users.get.validate_user_id", http.StatusBadRequest, errors.New("invalid user id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), organizationID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrUserNotFound):
			status = http.StatusNotFound
			c.JSON(status, gin.H{"error": "user not found"})
		default:
			c.JSON(status, gin.H{"error": "could not fetch user"})
		}
		logger.LogHandlerError(c, "users.get", status, err)
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:             user.ID,
		Email:          user.Email,
		OrganizationID: user.OrganizationID,
		CreatedAt:      user.CreatedAt,
	})
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
