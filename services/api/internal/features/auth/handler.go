package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	// spawn runs background work (the forgot-password lookup) so response time does not reveal
	// whether an account exists. Tests replace it with a synchronous runner.
	spawn func(func())
}

func NewHandler(service *Service, spawn func(func())) *Handler {
	if spawn == nil {
		spawn = func(f func()) { go f() }
	}
	return &Handler{service: service, spawn: spawn}
}

func fail(c *gin.Context, op string, status int, msg string, err error) {
	if err != nil {
		logger.LogHandlerError(c, op, status, err)
	}
	c.JSON(status, gin.H{"error": msg})
}

func (h *Handler) authResponse(c *gin.Context, status int, tokens Tokens, me MeResponse) {
	c.Header("Cache-Control", "no-store")
	c.JSON(status, AuthResponse{
		Token:        tokens.AccessToken,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         me,
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.register.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	tokens, me, err := h.service.Register(c.Request.Context(), req.Email, req.Password, req.BirthDate, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			fail(c, "auth.register", http.StatusConflict, "email already exists", nil)
		case errors.Is(err, ErrInvalidEmail), errors.Is(err, ErrWeakPassword), errors.Is(err, ErrInvalidBirthDate):
			fail(c, "auth.register", http.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, ErrUnderage), errors.Is(err, ErrBirthDateFuture), errors.Is(err, ErrBirthDateAbsurd):
			fail(c, "auth.register", http.StatusUnprocessableEntity, err.Error(), nil)
		default:
			fail(c, "auth.register", http.StatusInternalServerError, "could not create user", err)
		}
		return
	}
	h.authResponse(c, http.StatusCreated, tokens, me)
	logger.LogHandlerEvent(c, "auth.register.success", http.StatusCreated, map[string]string{"created_user_id": me.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.login.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	tokens, me, err := h.service.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			fail(c, "auth.login", http.StatusUnauthorized, "invalid credentials", nil)
			return
		}
		fail(c, "auth.login", http.StatusInternalServerError, "could not login", err)
		return
	}
	h.authResponse(c, http.StatusOK, tokens, me)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.refresh.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	tokens, me, err := h.service.Refresh(c.Request.Context(), req.RefreshToken, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			fail(c, "auth.refresh", http.StatusUnauthorized, "invalid refresh token", nil)
			return
		}
		fail(c, "auth.refresh", http.StatusInternalServerError, "could not refresh session", err)
		return
	}
	h.authResponse(c, http.StatusOK, tokens, me)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.logout.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.service.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			fail(c, "auth.logout", http.StatusBadRequest, "invalid request", nil)
			return
		}
		fail(c, "auth.logout", http.StatusInternalServerError, "could not logout", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Forgot(c *gin.Context) {
	var req ForgotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.forgot.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	email := req.Email
	h.spawn(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		h.service.Forgot(ctx, email)
	})
	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
}

func (h *Handler) Reset(c *gin.Context) {
	var req ResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, "auth.reset.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.service.Reset(c.Request.Context(), req.Token, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrWeakPassword):
			fail(c, "auth.reset", http.StatusBadRequest, err.Error(), nil)
		case errors.Is(err, ErrInvalidResetToken):
			fail(c, "auth.reset", http.StatusBadRequest, "invalid or expired reset token", nil)
		default:
			fail(c, "auth.reset", http.StatusInternalServerError, "could not reset password", err)
		}
		return
	}
	c.Status(http.StatusNoContent)
}
