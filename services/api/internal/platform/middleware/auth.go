package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// SessionChecker confirms that the session referenced by an access token is
// still active (not revoked by logout, password reset or account deletion).
type SessionChecker func(ctx context.Context, sessionID, userID string) (bool, error)

// AccessClaims are the claims carried by access tokens.
type AccessClaims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// ParseAccessToken strictly validates an HS256 access token.
func ParseAccessToken(secret []byte, tokenString string) (AccessClaims, error) {
	claims := AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid {
		return AccessClaims{}, errors.New("invalid token")
	}
	if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.SessionID) == "" {
		return AccessClaims{}, errors.New("invalid token claims")
	}
	return claims, nil
}

func RequireUser(secret []byte, checkSession SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			unauthorized(c, "missing token")
			return
		}
		claims, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			unauthorized(c, "invalid token")
			return
		}
		if checkSession != nil {
			active, err := checkSession(c.Request.Context(), claims.SessionID, claims.UserID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "could not verify session", "code": "session_check_failed"})
				return
			}
			if !active {
				unauthorized(c, "session expired")
				return
			}
		}
		c.Set("userID", claims.UserID)
		c.Set("sessionID", claims.SessionID)
		c.Next()
	}
}

func unauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": message, "code": "unauthorized"})
}

func bearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
