package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TokenIssuer is the `iss` claim of every access token this API signs.
const TokenIssuer = "go-react-saas-api"

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrSessionInactive  = errors.New("session is not active")
	ErrAccountSuspended = errors.New("account suspended")
)

// SessionChecker verifies that a session is still live and its user may act.
// It must return ErrSessionInactive or ErrAccountSuspended to signal refusal.
type SessionChecker func(ctx context.Context, userID, sessionID string) error

type Identity struct {
	UserID    string
	SessionID string
}

// ParseAccessToken strictly validates an HS256 access token and checks the session.
func ParseAccessToken(ctx context.Context, secret []byte, tokenString string, check SessionChecker) (Identity, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuer(TokenIssuer))
	if err != nil || !token.Valid {
		return Identity{}, ErrInvalidToken
	}
	userID, ok := stringClaim(claims, "uid")
	if !ok {
		return Identity{}, ErrInvalidToken
	}
	sessionID, ok := stringClaim(claims, "sid")
	if !ok {
		return Identity{}, ErrInvalidToken
	}
	if check != nil {
		if err := check(ctx, userID, sessionID); err != nil {
			return Identity{}, err
		}
	}
	return Identity{UserID: userID, SessionID: sessionID}, nil
}

func RequireUser(secret []byte, check SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token", "code": "missing_token"})
			return
		}
		id, err := ParseAccessToken(c.Request.Context(), secret, tokenString, check)
		switch {
		case err == nil:
		case errors.Is(err, ErrAccountSuspended):
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account suspended", "code": "account_suspended"})
			return
		case errors.Is(err, ErrSessionInactive), errors.Is(err, ErrInvalidToken):
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "code": "invalid_token"})
			return
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error", "code": "internal_error"})
			return
		}
		c.Set("userID", id.UserID)
		c.Set("sessionID", id.SessionID)
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
