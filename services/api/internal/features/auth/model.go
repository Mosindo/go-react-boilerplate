package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User is the auth-internal view of an account.
type User struct {
	ID    string
	Email string
}

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,max=254"`
	Password  string `json:"password" binding:"required,max=512"`
	BirthDate string `json:"birthDate" binding:"required,max=10"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,max=512"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotRequest struct {
	Email string `json:"email" binding:"required,max=254"`
}

type ResetRequest struct {
	Token    string `json:"token" binding:"required,max=512"`
	Password string `json:"password" binding:"required,max=512"`
}

// MeResponse is the contract's Me shape; it is shared with the account feature.
type MeResponse struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	BirthDate       string    `json:"birthDate"`
	Age             int       `json:"age"`
	CreatedAt       time.Time `json:"createdAt"`
	ProfileComplete bool      `json:"profileComplete"`
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
