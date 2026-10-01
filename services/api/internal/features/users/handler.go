package users

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password)
	switch {
	case err == nil:
		httpx.NoContent(c)
	case errors.Is(err, ErrWrongPassword):
		httpx.Error(c, http.StatusForbidden, "password is incorrect")
	case errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusNotFound, "user not found")
	default:
		logger.LogHandlerError(c, "users.delete_account", http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "could not delete account")
	}
}
