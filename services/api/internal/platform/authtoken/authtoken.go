// Package authtoken signs and strictly validates the API's HS256 JWTs.
package authtoken

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer = "lueur-api"

	// TypeAccess is the bearer token used on REST calls.
	TypeAccess = "access"
	// TypeRealtime is a short-lived ticket only accepted by the WebSocket endpoint.
	TypeRealtime = "realtime"

	AccessTTL   = 15 * time.Minute
	RealtimeTTL = 60 * time.Second
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	Type      string `json:"typ"`
	jwt.RegisteredClaims
}

// Sign issues a token of the given type for a user session.
func Sign(secret []byte, tokenType, userID, sessionID string, ttl time.Duration, now time.Time) (string, error) {
	if len(secret) == 0 || userID == "" || sessionID == "" {
		return "", errors.New("authtoken: missing signing input")
	}
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		Type:      tokenType,
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

// Parse validates signature, algorithm, issuer, expiration and token type.
func Parse(secret []byte, tokenString, expectedType string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(
		strings.TrimSpace(tokenString),
		&claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method type")
			}
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}
	if claims.Type != expectedType || strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.SessionID) == "" {
		return Claims{}, ErrInvalidToken
	}
	if claims.Subject != claims.UserID {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
