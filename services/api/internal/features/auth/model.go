package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID             string
	Email          string
	OrganizationID string
	CreatedAt      time.Time
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
	Password string `json:"password" binding:"required,max=128"`
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

type PasswordResetRequest struct {
	Email string `json:"email" binding:"required,max=320"`
}

type PasswordResetConfirm struct {
	Token       string `json:"token" binding:"required,max=512"`
	NewPassword string `json:"newPassword" binding:"required,max=128"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type MeResponse struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	OrganizationID string    `json:"organizationId"`
	CreatedAt      time.Time `json:"createdAt"`
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
	UserID         string `json:"uid"`
	OrganizationID string `json:"oid"`
	SessionID      string `json:"sid"`
	jwt.RegisteredClaims
}
