package auth

import (
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,max=72"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,max=254"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,max=254"`
	Code        string `json:"code" binding:"required,max=32"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=72"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=72"`
}

type MeResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type AuthResponse struct {
	AccessToken  string     `json:"accessToken"`
	RefreshToken string     `json:"refreshToken"`
	User         MeResponse `json:"user"`
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
}

type AccessTokenClaims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// cleanEmail trims and lowercases an address and checks that it is a plain
// address (no display name) with a dotted domain.
func cleanEmail(raw string) (string, bool) {
	email := strings.ToLower(strings.TrimSpace(raw))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", false
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || !strings.Contains(email[at+1:], ".") {
		return "", false
	}
	return email, true
}
