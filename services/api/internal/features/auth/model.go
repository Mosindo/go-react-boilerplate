package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
)

type User struct {
	ID         string
	Email      string
	HasProfile bool
	CreatedAt  time.Time
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
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	HasProfile bool      `json:"hasProfile"`
	CreatedAt  time.Time `json:"createdAt"`
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

type AccessTokenClaims = middleware.AccessClaims

func toMeResponse(u User) MeResponse {
	return MeResponse{ID: u.ID, Email: u.Email, HasProfile: u.HasProfile, CreatedAt: u.CreatedAt}
}
