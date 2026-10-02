package auth

import "time"

type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,max=128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,max=254"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,max=254"`
	Code        string `json:"code" binding:"required,max=32"`
	NewPassword string `json:"newPassword" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=128"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type AuthResponse struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	User         UserResponse `json:"user"`
}
