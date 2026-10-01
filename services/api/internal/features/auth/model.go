package auth

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Email trims surrounding whitespace while decoding, so validation sees the normalised value.
type Email string

func (e *Email) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*e = Email(strings.TrimSpace(s))
	return nil
}

type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

type RegisterRequest struct {
	Email    Email  `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    Email  `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,max=72"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotRequest struct {
	Email Email `json:"email" binding:"required,email,max=254"`
}

type ResetRequest struct {
	Email       Email  `json:"email" binding:"required,email,max=254"`
	Code        string `json:"code" binding:"required,max=32"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=72"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=72"`
}

type MeResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
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
