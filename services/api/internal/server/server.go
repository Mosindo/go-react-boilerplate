// Package server wires the HTTP router: platform middleware plus every feature
// module (handler -> service -> repository).
package server

import (
	"context"
	"net/url"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/matching"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/features/safety"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	Pool           *pgxpool.Pool
	JWTSecret      []byte
	AllowedOrigins []string
	TrustedProxies []string
	Store          storage.Store
	Mailer         auth.Mailer
	Hub            *realtime.Hub
	// Limits overrides the default abuse-protection budgets (tests only).
	Limits Limits
	// BcryptCost overrides the password hashing cost (tests only; 0 = default).
	BcryptCost int
}

// Limits are per-client request budgets, expressed per minute with a burst.
type Limits struct {
	AuthPerMinute, AuthBurst         int
	RecoveryPerMinute, RecoveryBurst int
}

func (l Limits) withDefaults() Limits {
	if l.AuthPerMinute == 0 {
		l.AuthPerMinute, l.AuthBurst = 30, 10
	}
	if l.RecoveryPerMinute == 0 {
		l.RecoveryPerMinute, l.RecoveryBurst = 6, 5
	}
	return l
}

func New(d Deps) (*gin.Engine, error) {
	r := gin.New()
	// Without an explicit allow-list gin would trust X-Forwarded-For from anyone,
	// letting clients spoof their IP and dodge per-IP rate limits.
	if err := r.SetTrustedProxies(d.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(d.AllowedOrigins))
	r.Use(middleware.BodyLimit(1 << 20))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	sessionLive := func(ctx context.Context, userID, sessionID string) (bool, error) {
		var live bool
		err := d.Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sessions
				WHERE id::text = $1 AND user_id::text = $2 AND revoked_at IS NULL AND expires_at > NOW())`,
			sessionID, userID).Scan(&live)
		return live, err
	}
	requireUser := middleware.RequireUser(d.JWTSecret, sessionLive)

	authService := auth.NewService(auth.NewPGRepository(d.Pool), d.JWTSecret, d.Mailer)
	profilesService := profiles.NewService(profiles.NewPGRepository(d.Pool), d.Store)
	if d.BcryptCost > 0 {
		authService.SetBcryptCost(d.BcryptCost)
	}
	authService.AddDeletionHook(profilesService)

	matchingService := matching.NewService(matching.NewPGRepository(d.Pool), d.Hub)
	chatService := chat.NewService(chat.NewPGRepository(d.Pool), d.Hub)
	safetyService := safety.NewService(safety.NewPGRepository(d.Pool), d.Hub)
	notificationsService := notifications.NewService(notifications.NewPGRepository(d.Pool))

	r.GET("/health", platformhandlers.NewHealthHandler(d.Pool))
	limits := d.Limits.withDefaults()
	auth.RegisterRoutes(r, auth.NewHandler(authService), requireUser,
		middleware.RateLimitByIP(limits.AuthPerMinute, limits.AuthBurst),
		middleware.RateLimitByIP(limits.RecoveryPerMinute, limits.RecoveryBurst))
	profiles.RegisterRoutes(r, profiles.NewHandler(profilesService), requireUser)
	matching.RegisterRoutes(r, matching.NewHandler(matchingService), requireUser)
	chat.RegisterRoutes(r, chat.NewHandler(chatService), requireUser)
	safety.RegisterRoutes(r, safety.NewHandler(safetyService), requireUser)
	notifications.RegisterRoutes(r, notifications.NewHandler(notificationsService), requireUser)

	r.GET("/ws", middleware.RateLimitByIP(60, 20), d.Hub.Handler(func(token string) (string, error) {
		claims, err := middleware.ParseAccessToken(d.JWTSecret, token)
		if err != nil {
			return "", err
		}
		live, err := sessionLive(context.Background(), claims.UserID, claims.SessionID)
		if err != nil || !live {
			return "", middleware.ErrInvalidToken
		}
		return claims.UserID, nil
	}, originHosts(d.AllowedOrigins)))
	return r, nil
}

// originHosts converts allowed CORS origins (https://app.example.com) into the
// host patterns the WebSocket origin check expects (app.example.com).
func originHosts(origins []string) []string {
	hosts := make([]string, 0, len(origins))
	for _, o := range origins {
		if u, err := url.Parse(o); err == nil && u.Host != "" {
			hosts = append(hosts, u.Host)
		}
	}
	return hosts
}
