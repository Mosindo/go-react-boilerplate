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

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "password confirmation required", "users.delete.bind", err)
		return
	}
	userID := httpx.UserID(c)
	if err := h.service.DeleteAccount(c.Request.Context(), userID, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httpx.Fail(c, http.StatusForbidden, "invalid_credentials", "password is incorrect", "users.delete", err)
		case errors.Is(err, ErrUserNotFound):
			httpx.Fail(c, http.StatusNotFound, "not_found", "account not found", "users.delete", err)
		default:
			httpx.Fail(c, http.StatusInternalServerError, "internal", "could not delete account", "users.delete", err)
		}
		return
	}
	logger.LogHandlerEvent(c, "users.delete.success", http.StatusNoContent, map[string]string{"deleted_user_id": userID})
	c.Status(http.StatusNoContent)
}
