// Package jwtauth signs and strictly verifies access tokens (HS256 only).
package jwtauth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims are the only custom claims of an access token. There is no tenant claim.
type Claims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

var ErrInvalidToken = errors.New("invalid token")

// Sign creates an HS256 token that always carries iat and exp.
func Sign(secret []byte, userID, sessionID string, ttl time.Duration, now time.Time) (string, error) {
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// Parse verifies signature (HS256 only, never "none" or another HMAC size),
// mandatory expiry and the presence of uid/sid.
func Parse(secret []byte, tokenString string) (Claims, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || token == nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}
	if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.SessionID) == "" {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
