package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User is the account as seen by its owner.
type User struct {
	ID         string
	Email      string
	BirthDate  time.Time
	CreatedAt  time.Time
	HasProfile bool
	PhotoCount int
}

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	BirthDate string `json:"birthDate"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Code     string `json:"code"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type DeleteAccountRequest struct {
	Password string `json:"password"`
}

type MeResponse struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	Age        int       `json:"age"`
	CreatedAt  time.Time `json:"createdAt"`
	HasProfile bool      `json:"hasProfile"`
	PhotoCount int       `json:"photoCount"`
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
