package auth

import "time"

type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

type DeleteAccountRequest struct {
	Password string `json:"password"`
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

type OnboardingState struct {
	HasProfile bool `json:"hasProfile"`
	PhotoCount int  `json:"photoCount"`
	Completed  bool `json:"completed"`
}

type MeResponse struct {
	UserResponse
	Onboarding OnboardingState `json:"onboarding"`
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
}
