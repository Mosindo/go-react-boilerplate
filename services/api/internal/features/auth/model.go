package auth

import (
	"time"
)

type User struct {
	ID               string
	Email            string
	ProfileCompleted bool
	CreatedAt        time.Time
}

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"`
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
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	ProfileCompleted bool      `json:"profileCompleted"`
	CreatedAt        time.Time `json:"createdAt"`
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

func toMe(u User) MeResponse {
	return MeResponse{ID: u.ID, Email: u.Email, ProfileCompleted: u.ProfileCompleted, CreatedAt: u.CreatedAt}
}
