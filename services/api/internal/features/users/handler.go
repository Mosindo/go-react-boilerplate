package users

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.ListContacts(c.Request.Context(), httpx.UserID(c),
		httpx.Limit(c.Query("limit"), 20, 100), httpx.Offset(c.Query("offset"), 10_000))
	if err != nil {
		httpx.Internal(c, "users.list", err)
		return
	}
	c.JSON(http.StatusOK, UsersResponse{Users: list})
}

func (h *Handler) GetByID(c *gin.Context) {
	id := strings.ToLower(strings.TrimSpace(c.Param("userId")))
	if !httpx.IsUUID(id) {
		httpx.BadRequest(c, "invalid_user_id", "invalid user id")
		return
	}
	u, err := h.service.GetVisible(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, "users.get", http.StatusNotFound, "user_not_found", "user not found", nil)
			return
		}
		httpx.Internal(c, "users.get", err)
		return
	}
	c.JSON(http.StatusOK, u)
}
