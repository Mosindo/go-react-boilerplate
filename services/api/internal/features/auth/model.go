package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"createdAt"`
	ProfileComplete bool      `json:"profileComplete"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
	Password string `json:"password" binding:"required,max=128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ResetRequest struct {
	Email string `json:"email" binding:"required,max=320"`
}

type ResetConfirmRequest struct {
	Token    string `json:"token" binding:"required,max=64"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type AuthResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         User   `json:"user"`
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
