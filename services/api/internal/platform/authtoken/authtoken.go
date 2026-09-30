// Package authtoken signs and strictly validates the HS256 JWTs used by the
// API: short-lived access tokens and one-minute realtime connection tickets.
// The "typ" claim prevents a ticket from being replayed as an access token
// and vice versa.
package authtoken

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TypeAccess = "access"
	TypeTicket = "ws_ticket"

	AccessTTL = 15 * time.Minute
	TicketTTL = time.Minute
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid,omitempty"`
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
	return m.sign(Claims{UserID: userID, SessionID: sessionID, Type: TypeAccess}, AccessTTL)
}

func (m *Manager) SignTicket(userID string) (string, error) {
	return m.sign(Claims{UserID: userID, Type: TypeTicket}, TicketTTL)
}

func (m *Manager) sign(claims Claims, ttl time.Duration) (string, error) {
	now := m.now()
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse validates signature, algorithm, expiry and expected token type.
func (m *Manager) Parse(tokenString, expectedType string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("unexpected signing method")
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}
	if claims.Type != expectedType || strings.TrimSpace(claims.UserID) == "" {
		return Claims{}, ErrInvalidToken
	}
	if expectedType == TypeAccess && strings.TrimSpace(claims.SessionID) == "" {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}
