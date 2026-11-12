package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

// Identity is the authenticated principal extracted from a valid access token.
type Identity struct {
	UserID         string
	OrganizationID string
	SessionID      string
}

// SessionChecker reports whether a session is still active (not revoked or
// expired, user not deleted). It lets logout, password reset and account
// deletion invalidate access tokens immediately instead of after their TTL.
type SessionChecker func(ctx context.Context, sessionID, userID string) (bool, error)

// TokenVerifier validates HS256 access tokens with strict claim checks.
type TokenVerifier struct {
	secret       []byte
	checkSession SessionChecker
}

func NewTokenVerifier(secret []byte, checkSession SessionChecker) *TokenVerifier {
	return &TokenVerifier{secret: secret, checkSession: checkSession}
}

func (v *TokenVerifier) Verify(ctx context.Context, tokenString string) (Identity, error) {
	if strings.TrimSpace(tokenString) == "" {
		return Identity{}, ErrInvalidToken
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method type")
		}
		return v.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid {
		return Identity{}, ErrInvalidToken
	}

	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil || expiresAt.Before(time.Now()) {
		return Identity{}, ErrInvalidToken
	}

	identity := Identity{}
	var ok bool
	if identity.UserID, ok = stringClaim(claims, "uid"); !ok {
		return Identity{}, ErrInvalidToken
	}
	if identity.OrganizationID, ok = stringClaim(claims, "oid"); !ok {
		return Identity{}, ErrInvalidToken
	}
	if identity.SessionID, ok = stringClaim(claims, "sid"); !ok {
		return Identity{}, ErrInvalidToken
	}

	if v.checkSession != nil {
		active, err := v.checkSession(ctx, identity.SessionID, identity.UserID)
		if err != nil {
			return Identity{}, err
		}
		if !active {
			return Identity{}, ErrInvalidToken
		}
	}

	return identity, nil
}

// StillActive re-checks the session behind an identity (used by long-lived
// connections such as WebSockets, whose access token may have expired since).
func (v *TokenVerifier) StillActive(ctx context.Context, identity Identity) bool {
	if v.checkSession == nil {
		return true
	}
	active, err := v.checkSession(ctx, identity.SessionID, identity.UserID)
	return err == nil && active
}

// RequireUser authenticates the request with a Bearer access token.
func RequireUser(verifier *TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		identity, err := verifier.Verify(c.Request.Context(), tokenString)
		if err != nil {
			if errors.Is(err, ErrInvalidToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
				return
			}
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "could not verify session"})
			return
		}

		c.Set("userID", identity.UserID)
		c.Set("organizationID", identity.OrganizationID)
		c.Set("sessionID", identity.SessionID)
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
