package auth

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func authResponse(tokens Tokens, user User) AuthResponse {
	return AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         UserResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Register(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, authResponse(tokens, user))
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	tokens, user, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, authResponse(tokens, user))
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusAccepted)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), req.Email, req.Code, req.Password); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Me(c *gin.Context) {
	user, state, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, MeResponse{
		UserResponse: UserResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
		Onboarding:   state,
	})
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	var req DeleteAccountRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), httpx.UserID(c), req.Password); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}
