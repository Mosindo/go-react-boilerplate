// Package account owns account-level operations that span several features.
package account

import (
	"context"
	"errors"
	"net/http"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type PasswordVerifier interface {
	DeleteAccount(ctx context.Context, userID, password string) error
}

type PhotoFiles interface {
	StorageKeys(ctx context.Context, userID string) ([]string, error)
	RemoveFiles(keys []string)
}

type Handler struct {
	accounts PasswordVerifier
	photos   PhotoFiles
}

func NewHandler(accounts PasswordVerifier, photos PhotoFiles) *Handler {
	return &Handler{accounts: accounts, photos: photos}
}

type DeleteRequest struct {
	Password string `json:"password" binding:"required,max=72"`
}

// Delete permanently removes the account. Rows owned by the user disappear
// through foreign-key cascades (profile, swipes, matches, messages, ...);
// photo files are removed from storage afterwards.
func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "account.delete.bind", err)
		return
	}
	userID := c.GetString("userID")

	keys, err := h.photos.StorageKeys(c.Request.Context(), userID)
	if err != nil {
		httpx.Internal(c, "account.delete.keys", err)
		return
	}
	switch err := h.accounts.DeleteAccount(c.Request.Context(), userID, req.Password); {
	case err == nil:
	case errors.Is(err, auth.ErrInvalidCredentials):
		httpx.Fail(c, "account.delete", http.StatusForbidden, "password is incorrect", nil)
		return
	case errors.Is(err, auth.ErrUserNotFound):
		httpx.Fail(c, "account.delete", http.StatusNotFound, "user not found", nil)
		return
	default:
		httpx.Internal(c, "account.delete", err)
		return
	}
	h.photos.RemoveFiles(keys)
	c.Status(http.StatusNoContent)
}
