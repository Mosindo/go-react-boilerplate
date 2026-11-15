package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string
	Email     string
	IsDemo    bool
	CreatedAt time.Time
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	// AcceptTerms confirms the user is an adult and accepts the rules.
	AcceptTerms bool `json:"acceptTerms"`
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

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,max=320"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,max=320"`
	Code        string `json:"code" binding:"required,len=6,numeric"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=128"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=128"`
	NewPassword     string `json:"newPassword" binding:"required,min=8,max=128"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type MeResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	IsDemo    bool      `json:"isDemo"`
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

func toMeResponse(user User) MeResponse {
	return MeResponse{ID: user.ID, Email: user.Email, IsDemo: user.IsDemo, CreatedAt: user.CreatedAt}
}
