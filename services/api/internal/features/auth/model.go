package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

// bcrypt only hashes the first 72 bytes, so longer passwords are rejected
// instead of being silently truncated.
const (
	minPasswordLength = 10
	maxPasswordBytes  = 72
)

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=400"`
	Password string `json:"password" binding:"required,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=400"`
	Password string `json:"password" binding:"required,max=72"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,max=400"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required,max=256"`
	Password string `json:"password" binding:"required,max=72"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=72"`
	NewPassword     string `json:"newPassword" binding:"required,max=72"`
}

type MeResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type AuthResponse struct {
	Token        string     `json:"token"`
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
