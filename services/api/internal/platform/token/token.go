// Package token issues and validates the HS256 access tokens used by the API.
// Validation is strict: algorithm, issuer, audience, expiration and subject
// claims are all mandatory.
package token

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer   = "lueur-api"
	Audience = "lueur-app"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret []byte, ttl time.Duration) *Manager {
	return &Manager{secret: secret, ttl: ttl}
}

func (m *Manager) Sign(userID, sessionID string, now time.Time) (string, error) {
	claims := Claims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse validates a token and returns its subject (user id) and session id.
func (m *Manager) Parse(raw string) (userID string, sessionID string, err error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(Audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !parsed.Valid {
		return "", "", ErrInvalidToken
	}
	if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(claims.SessionID) == "" {
		return "", "", ErrInvalidToken
	}
	return claims.Subject, claims.SessionID, nil
}
