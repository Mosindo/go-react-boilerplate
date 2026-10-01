package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID             string
	Email          string
	OrganizationID string
	Role           string
	HasProfile     bool
	PhotoCount     int
	CreatedAt      time.Time
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,max=72"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required,max=512"`
}

type MeResponse struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	OrganizationID  string    `json:"organizationId"`
	Role            string    `json:"role"`
	HasProfile      bool      `json:"hasProfile"`
	PhotoCount      int       `json:"photoCount"`
	ProfileComplete bool      `json:"profileComplete"`
	CreatedAt       time.Time `json:"createdAt"`
}

func newMeResponse(u User) MeResponse {
	return MeResponse{
		ID: u.ID, Email: u.Email, OrganizationID: u.OrganizationID, Role: u.Role,
		HasProfile: u.HasProfile, PhotoCount: u.PhotoCount,
		ProfileComplete: u.HasProfile && u.PhotoCount > 0, CreatedAt: u.CreatedAt,
	}
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,max=254"`
}

type ResetPasswordRequest struct {
	Code        string `json:"code" binding:"required,max=64"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=72"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required,max=72"`
	NewPassword     string `json:"newPassword" binding:"required,min=8,max=72"`
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
