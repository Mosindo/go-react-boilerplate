package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SessionValidator confirms that the session referenced by an access token is still active
// (not revoked by logout, password reset or account deletion) and that the account may act.
type SessionValidator interface {
	ValidateSession(ctx context.Context, sessionID, userID string) (bool, error)
}

type AccessClaims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// ParseAccessToken strictly validates an HS256 access token and returns its claims.
func ParseAccessToken(secret []byte, tokenString string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
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
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.SessionID) == "" {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

func RequireUser(secret []byte, sessions SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			abortUnauthorized(c, "missing_token", "Authentification requise.")
			return
		}

		claims, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			abortUnauthorized(c, "invalid_token", "Session invalide ou expirée.")
			return
		}

		if sessions != nil {
			active, err := sessions.ValidateSession(c.Request.Context(), claims.SessionID, claims.UserID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Service momentanément indisponible.", "code": "session_check_failed"})
				return
			}
			if !active {
				abortUnauthorized(c, "session_revoked", "Session invalide ou expirée.")
				return
			}
		}

		c.Set("userID", claims.UserID)
		c.Set("sessionID", claims.SessionID)
		c.Next()
	}
}

func abortUnauthorized(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message, "code": code})
}

func bearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
