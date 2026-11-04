package auth

import (
	"errors"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// translate maps domain errors to application errors.
func translate(err error) error {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		return platformerrors.New(http.StatusBadRequest, "validation", ve.Message)
	case errors.Is(err, ErrEmailExists):
		return platformerrors.New(http.StatusConflict, "email_exists", "email already exists")
	case errors.Is(err, ErrInvalidCredentials):
		return platformerrors.New(http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
	case errors.Is(err, ErrInvalidRefreshToken):
		return platformerrors.New(http.StatusUnauthorized, "invalid_refresh_token", "invalid refresh token")
	case errors.Is(err, ErrUserNotFound):
		return platformerrors.New(http.StatusUnauthorized, "unauthorized", "user not found")
	case errors.Is(err, ErrInvalidResetCode):
		return platformerrors.New(http.StatusBadRequest, "invalid_code", "invalid or expired code")
	}
	return err
}

func respondAuth(c *gin.Context, status int, tokens Tokens, user User) {
	httpx.NoStore(c)
	c.JSON(status, AuthResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, User: user})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.register", translate(err))
		return
	}
	respondAuth(c, http.StatusCreated, tokens, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.login", translate(err))
		return
	}
	respondAuth(c, http.StatusOK, tokens, user)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, "auth.refresh", translate(err))
		return
	}
	respondAuth(c, http.StatusOK, tokens, user)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Fail(c, "auth.logout", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httpx.Fail(c, "auth.forgot", err)
		return
	}
	// Always 202: do not reveal whether the account exists.
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.NewPassword); err != nil {
		httpx.Fail(c, "auth.reset", translate(err))
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	user, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "auth.me", translate(err))
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, user)
}
