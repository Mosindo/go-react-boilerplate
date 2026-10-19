package middleware

import (
	"context"
	"errors"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

type AccessClaims struct {
	UserID    string
	SessionID string
}

// ParseAccessToken strictly validates an HS256 access token: signing method,
// signature, mandatory expiration and the custom claims.
func ParseAccessToken(secret []byte, tokenString string) (AccessClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method type")
		}
		return secret, nil
	}, jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return AccessClaims{}, ErrInvalidToken
	}

	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return AccessClaims{}, ErrInvalidToken
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return AccessClaims{}, ErrInvalidToken
	}
	return AccessClaims{UserID: userID, SessionID: sessionID}, nil
}

// SessionValidator reports whether the session behind an access token is still
// live (not logged out, reset, expired or deleted along with its account).
type SessionValidator func(ctx context.Context, userID, sessionID string) (bool, error)

// RequireUser authenticates the request. When validate is non-nil, the token's
// session must also still be active, so logout, password reset and account
// deletion take effect immediately instead of when the access token expires.
func RequireUser(secret []byte, validate SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.Abort()
			platformerrors.Respond(c, 401, "unauthorized", "missing token")
			return
		}
		claims, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			c.Abort()
			platformerrors.Respond(c, 401, "unauthorized", "invalid token")
			return
		}
		if validate != nil {
			live, err := validate(c.Request.Context(), claims.UserID, claims.SessionID)
			if err != nil {
				c.Abort()
				platformerrors.Respond(c, 503, "unavailable", "could not verify session")
				return
			}
			if !live {
				c.Abort()
				platformerrors.Respond(c, 401, "unauthorized", "session expired")
				return
			}
		}
		c.Set("userID", claims.UserID)
		c.Set("sessionID", claims.SessionID)
		c.Next()
	}
}

func bearerToken(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func stringClaim(claims jwt.MapClaims, key string) (string, bool) {
	value, ok := claims[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}
