package account

import (
	"errors"
	"net/http"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func fail(c *gin.Context, op string, status int, msg string, err error) {
	if err != nil {
		logger.LogHandlerError(c, op, status, err)
	}
	c.JSON(status, gin.H{"error": msg})
}

func (h *Handler) Me(c *gin.Context) {
	me, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			fail(c, "account.me", http.StatusUnauthorized, "account not found", nil)
			return
		}
		fail(c, "account.me", http.StatusInternalServerError, "could not load account", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, me)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "account.password.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	err := h.service.ChangePassword(c.Request.Context(), c.GetString("userID"), c.GetString("sessionID"), req.CurrentPassword, req.NewPassword)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, auth.ErrWeakPassword):
		fail(c, "account.password", http.StatusBadRequest, err.Error(), nil)
	case errors.Is(err, ErrWrongPassword):
		fail(c, "account.password", http.StatusForbidden, "current password is incorrect", nil)
	case errors.Is(err, ErrNotFound):
		fail(c, "account.password", http.StatusUnauthorized, "account not found", nil)
	default:
		fail(c, "account.password", http.StatusInternalServerError, "could not change password", err)
	}
}

func (h *Handler) Delete(c *gin.Context) {
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "account.delete.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	err := h.service.Delete(c.Request.Context(), c.GetString("userID"), req.Password)
	switch {
	case err == nil:
		logger.LogHandlerEvent(c, "account.deleted", http.StatusNoContent, nil)
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrWrongPassword):
		fail(c, "account.delete", http.StatusForbidden, "password is incorrect", nil)
	case errors.Is(err, ErrNotFound):
		fail(c, "account.delete", http.StatusUnauthorized, "account not found", nil)
	default:
		fail(c, "account.delete", http.StatusInternalServerError, "could not delete account", err)
	}
}
