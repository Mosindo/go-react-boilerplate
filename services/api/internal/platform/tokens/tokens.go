// Package tokens signs and validates the HS256 JWTs used by the API.
//
// Two token kinds exist and are never interchangeable:
//   - access tokens (typ "access", 15 min) sent as Bearer tokens;
//   - realtime tickets (typ "rt", 60 s) passed once when opening the WebSocket.
package tokens

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TypeAccess   = "access"
	TypeRealtime = "rt"

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

type Manager struct {
	secret []byte
	now    func() time.Time
}

func NewManager(secret []byte) *Manager {
	return &Manager{secret: secret, now: time.Now}
}

func (m *Manager) SignAccess(userID, sessionID string) (string, error) {
	return m.sign(userID, sessionID, TypeAccess, AccessTTL)
}

func (m *Manager) SignRealtimeTicket(userID, sessionID string) (string, error) {
	return m.sign(userID, sessionID, TypeRealtime, RealtimeTTL)
}

func (m *Manager) sign(userID, sessionID, kind string, ttl time.Duration) (string, error) {
	now := m.now()
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		Type:      kind,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse validates signature, algorithm, expiration and the expected token type.
func (m *Manager) Parse(raw, expectedType string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method type")
		}
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}
	if claims.Type != expectedType || strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.SessionID) == "" {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
