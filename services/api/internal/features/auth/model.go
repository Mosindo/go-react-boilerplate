package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string
	Email     string
	BirthDate time.Time
	CreatedAt time.Time
}

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email,max=254"`
	Password  string `json:"password" binding:"required,min=8,max=72"`
	BirthDate string `json:"birthDate" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,max=128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email,max=254"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email,max=254"`
	Code        string `json:"code" binding:"required,max=32"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=72"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type MeResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	BirthDate string    `json:"birthDate"`
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

func toMeResponse(u User) MeResponse {
	return MeResponse{ID: u.ID, Email: u.Email, BirthDate: u.BirthDate.Format("2006-01-02"), CreatedAt: u.CreatedAt}
}
