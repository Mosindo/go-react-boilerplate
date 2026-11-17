// Package authtoken signs and validates the short-lived HS256 access tokens used by the API.
package authtoken

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const Issuer = "lueur-api"

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// Sign issues an access token bound to a user and a refresh session.
func Sign(secret []byte, userID, sessionID string, ttl time.Duration, now time.Time) (string, error) {
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// Parse validates signature, algorithm, issuer, expiration and required custom claims.
func Parse(secret []byte, raw string) (Claims, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method type")
		}
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(Issuer),
	)
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}
	if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.SessionID) == "" || claims.Subject != claims.UserID {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
