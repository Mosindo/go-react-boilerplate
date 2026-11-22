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
	// The address is validated after trimming/lower-casing in the service.
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,max=128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
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

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,max=254"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Code     string `json:"code" binding:"required,len=6,numeric"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=128"`
	NewPassword     string `json:"newPassword" binding:"required,min=8,max=128"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type StoredResetToken struct {
	ID       string
	CodeHash string
	Attempts int
}

// DeletedAccount describes what must be cleaned up outside the database.
type DeletedAccount struct {
	UserID        string
	PhotoKeys     []string
	FormerMatches []string
}
