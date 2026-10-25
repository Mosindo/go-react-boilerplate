package auth

import (
	"encoding/json"
	"strings"
	"time"
)

type User struct {
	ID              string
	Email           string
	ProfileComplete bool
	CreatedAt       time.Time
}

const (
	MinPasswordLength = 8
	// MaxPasswordBytes is bcrypt's hard input limit.
	MaxPasswordBytes = 72
)

// Email trims and lower-cases itself while decoding, so validation sees the normalized address.
type Email string

func (e *Email) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*e = Email(strings.ToLower(strings.TrimSpace(raw)))
	return nil
}

type RegisterRequest struct {
	Email    Email  `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,max=128"`
}

type LoginRequest struct {
	Email    Email  `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,max=128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotPasswordRequest struct {
	Email Email `json:"email" binding:"required,email,max=254"`
}

type ResetPasswordRequest struct {
	Email    Email  `json:"email" binding:"required,email,max=254"`
	Code     string `json:"code" binding:"required,max=32"`
	Password string `json:"password" binding:"required,max=128"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=128"`
	NewPassword     string `json:"newPassword" binding:"required,max=128"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type MeResponse struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"createdAt"`
	ProfileComplete bool      `json:"profileComplete"`
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
