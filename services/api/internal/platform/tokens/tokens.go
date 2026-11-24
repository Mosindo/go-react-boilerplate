// Package tokens signs and validates the HS256 JWTs used by the API:
// short-lived access tokens and one-minute realtime connection tickets.
package tokens

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer           = "lueur-api"
	AudienceAccess   = "lueur-app"
	AudienceRealtime = "lueur-realtime"
)

var ErrInvalidToken = errors.New("invalid token")

// Claims carried by every token issued by the API.
type Claims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// Manager signs and parses tokens with a single shared secret.
type Manager struct {
	secret []byte
	now    func() time.Time
}

func NewManager(secret []byte) *Manager {
	return &Manager{secret: secret, now: time.Now}
}

// Sign issues a token for audience that expires after ttl.
func (m *Manager) Sign(userID, sessionID, audience string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		return "", errors.New("token ttl must be positive")
	}
	now := m.now()
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse validates signature, algorithm, issuer, audience and expiration.
func (m *Manager) Parse(tokenString, audience string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(
		strings.TrimSpace(tokenString),
		&claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, ErrInvalidToken
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}
	if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.SessionID) == "" || claims.Subject != claims.UserID {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
