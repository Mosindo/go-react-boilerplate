// Package token signs and verifies the short-lived HS256 JWTs used by the API.
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TypeAccess = "access"
	TypeWS     = "ws"
)

type Claims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	Type      string `json:"typ"`
	jwt.RegisteredClaims
}

var ErrInvalid = errors.New("invalid token")

func Sign(secret []byte, userID, sessionID, typ string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		Type:      typ,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// Parse enforces HS256, a mandatory expiry, the expected token type and non-empty identity claims.
func Parse(secret []byte, raw, wantType string) (Claims, error) {
	var claims Claims
	tok, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !tok.Valid {
		return Claims{}, ErrInvalid
	}
	if claims.Type != wantType || claims.UserID == "" || claims.SessionID == "" {
		return Claims{}, ErrInvalid
	}
	return claims, nil
}
