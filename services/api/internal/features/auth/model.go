package auth

import "time"

type User struct {
	ID              string
	Email           string
	IsDemo          bool
	ProfileComplete bool
	CreatedAt       time.Time
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
	Password string `json:"password" binding:"required,max=128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,max=320"`
}

type ResetPasswordRequest struct {
	Email    string `json:"email" binding:"required,max=320"`
	Code     string `json:"code" binding:"required,max=32"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=128"`
	NewPassword     string `json:"newPassword" binding:"required,min=8,max=128"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,max=128"`
}

type MeResponse struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	IsDemo          bool      `json:"isDemo"`
	ProfileComplete bool      `json:"profileComplete"`
	CreatedAt       time.Time `json:"createdAt"`
}

type AuthResponse struct {
	AccessToken  string     `json:"accessToken"`
	RefreshToken string     `json:"refreshToken"`
	User         MeResponse `json:"user"`
}

type RealtimeTicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expiresIn"`
}

func toMeResponse(user User) MeResponse {
	return MeResponse{
		ID:              user.ID,
		Email:           user.Email,
		IsDemo:          user.IsDemo,
		ProfileComplete: user.ProfileComplete,
		CreatedAt:       user.CreatedAt,
	}
}
