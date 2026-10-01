package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionInfo is what the API needs to know about the session behind a token.
type SessionInfo struct {
	Active bool
	Status string
	Role   string
}

// SessionChecker lets RequireUser verify server-side that a still-unexpired
// access token belongs to a live session and an account in good standing.
type SessionChecker interface {
	CheckSession(ctx context.Context, sessionID, userID string) (SessionInfo, error)
}

// PGSessionChecker implements SessionChecker with one indexed query.
type PGSessionChecker struct {
	pool *pgxpool.Pool
}

func NewPGSessionChecker(pool *pgxpool.Pool) *PGSessionChecker {
	return &PGSessionChecker{pool: pool}
}

func (p *PGSessionChecker) CheckSession(ctx context.Context, sessionID, userID string) (SessionInfo, error) {
	var info SessionInfo
	err := p.pool.QueryRow(ctx, `
		SELECT TRUE, u.status, u.role
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = $1
		  AND s.user_id = $2
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()
	`, sessionID, userID).Scan(&info.Active, &info.Status, &info.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionInfo{}, nil
		}
		return SessionInfo{}, err
	}
	return info, nil
}

// ParseAccessToken validates an HS256 token and returns its claims.
func ParseAccessToken(secret []byte, tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method type")
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		return nil, errors.New("missing expiration")
	}
	return claims, nil
}

// RequireUser authenticates the bearer token. When checker is non-nil the
// session must still be active and the account must not be suspended.
func RequireUser(secret []byte, checker SessionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		claims, err := ParseAccessToken(secret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		if aud, _ := claims.GetAudience(); len(aud) > 0 {
			// Tokens minted for another purpose (e.g. websocket tickets) are not access tokens.
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		userID, ok := stringClaim(claims, "uid")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		sessionID, ok := stringClaim(claims, "sid")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		organizationID, _ := stringClaim(claims, "oid")

		role := "user"
		if checker != nil {
			info, err := checker.CheckSession(c.Request.Context(), sessionID, userID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "could not verify session"})
				return
			}
			if !info.Active {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
				return
			}
			if info.Status != "active" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account suspended"})
				return
			}
			role = info.Role
		}

		c.Set("userID", userID)
		c.Set("organizationID", organizationID)
		c.Set("sessionID", sessionID)
		c.Set("userRole", role)
		c.Next()
	}
}

// RequireAdmin must run after RequireUser.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("userRole") != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
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
