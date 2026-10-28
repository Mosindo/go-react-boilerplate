package auth

import "time"

type User struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	CreatedAt       time.Time `json:"createdAt"`
	ProfileComplete bool      `json:"profileComplete"`
}

type Credentials struct {
	Email    string `json:"email" binding:"required,max=320"`
	Password string `json:"password" binding:"required,max=128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,max=320"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
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

type AuthResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         User   `json:"user"`
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
}
