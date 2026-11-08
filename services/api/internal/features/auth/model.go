package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type StoredUser struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

type StoredUserAuth struct {
	StoredUser
	PasswordHash string
}

type StoredSession struct {
	ID     string
	UserID string
}

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	BirthDate string `json:"birthDate"` // YYYY-MM-DD
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type DeleteAccountRequest struct {
	Password string `json:"password"`
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
