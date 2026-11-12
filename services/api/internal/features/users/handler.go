package users

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
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
	list, err := h.service.List(c.Request.Context(), httpx.OrganizationID(c), httpx.UserID(c), httpx.Limit(c.Query("limit"), 20, 100), httpx.Offset(c.Query("offset"), 10000))
	if err != nil {
		logger.LogHandlerError(c, "users.list", http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not fetch users")
		return
	}
	c.JSON(http.StatusOK, UsersResponse{Users: list})
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("userId")
	if !httpx.IsUUID(id) {
		httpx.Fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	u, err := h.service.Get(c.Request.Context(), httpx.OrganizationID(c), httpx.UserID(c), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httpx.Fail(c, http.StatusNotFound, "user not found")
			return
		}
		logger.LogHandlerError(c, "users.get", http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not fetch user")
		return
	}
	c.JSON(http.StatusOK, u)
}
