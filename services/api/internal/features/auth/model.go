package auth

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

// Email trims and lower-cases itself while decoding JSON, so validation sees the normalized value.
type Email string

func (e *Email) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*e = Email(strings.ToLower(strings.TrimSpace(raw)))
	return nil
}

type RegisterRequest struct {
	Email    Email  `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type LoginRequest struct {
	Email    Email  `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,max=128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotPasswordRequest struct {
	Email Email `json:"email" binding:"required,email,max=254"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required,max=256"`
	Password string `json:"password" binding:"required,min=8,max=128"`
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
