package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
	// Onboarding progress, so clients know which step to show.
	HasProfile     bool
	HasPreferences bool
	HasPhotos      bool
}

func (u User) OnboardingComplete() bool {
	return u.HasProfile && u.HasPreferences && u.HasPhotos
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,max=128"`
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

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,max=254"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,max=254"`
	Code        string `json:"code" binding:"required,max=32"`
	NewPassword string `json:"newPassword" binding:"required,max=128"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=128"`
	NewPassword     string `json:"newPassword" binding:"required,max=128"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type MeResponse struct {
	ID                 string    `json:"id"`
	Email              string    `json:"email"`
	CreatedAt          time.Time `json:"createdAt"`
	HasProfile         bool      `json:"hasProfile"`
	HasPreferences     bool      `json:"hasPreferences"`
	HasPhotos          bool      `json:"hasPhotos"`
	OnboardingComplete bool      `json:"onboardingComplete"`
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

func toMeResponse(u User) MeResponse {
	return MeResponse{
		ID:                 u.ID,
		Email:              u.Email,
		CreatedAt:          u.CreatedAt,
		HasProfile:         u.HasProfile,
		HasPreferences:     u.HasPreferences,
		HasPhotos:          u.HasPhotos,
		OnboardingComplete: u.OnboardingComplete(),
	}
}
