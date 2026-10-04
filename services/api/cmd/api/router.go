package main

import (
	"fmt"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	discoveryfeature "example.com/api/internal/features/discovery"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Dependencies are the infrastructure pieces the HTTP layer is built on.
type Dependencies struct {
	Config config.Config
	DB     *pgxpool.Pool
	Files  storage.Store
	Mailer mail.Mailer
	// RelaxLimits multiplies every rate limit by 1000. Tests only.
	RelaxLimits bool
}

// newRouter wires every feature (handler -> service -> repository). It is the only place that
// knows about all features.
func newRouter(d Dependencies) (*gin.Engine, error) {
	cfg := d.Config
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("trusted proxies: %w", err)
	}
	r.MaxMultipartMemory = 8 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	origins := cfg.AllowedOrigins
	if len(origins) == 0 && !cfg.IsProduction() {
		origins = []string{"*"} // development: let `expo start --web` call the API from any localhost port
	}
	r.Use(middleware.CORS(origins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.NewRateLimiter(600*scaleFor(d), 200*scaleFor(d)).ByIP()) // coarse per-IP ceiling for every endpoint

	jwtSecret := []byte(cfg.JWTSecret)

	// Auth first: the session validator it provides guards every other route and the websocket.
	authRepo := authfeature.NewPGRepository(d.DB)
	authService := authfeature.NewService(authRepo, jwtSecret, authfeature.Deps{
		Mailer: d.Mailer, Files: d.Files, AppBaseURL: cfg.AppBaseURL,
	})
	requireUser := middleware.RequireUser(jwtSecret, authService.ValidateSession)
	hub := realtime.NewHub(jwtSecret, authService.ValidateSession)

	profilesRepo := profilesfeature.NewPGRepository(d.DB)
	profilesService := profilesfeature.NewService(profilesRepo)

	photosService := photosfeature.NewService(photosfeature.NewPGRepository(d.DB), d.Files)

	chatService := chatfeature.NewService(chatfeature.NewPGRepository(d.DB), hub)
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(d.DB, profilesRepo), chatService, hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(d.DB), hub)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(d.DB))

	scale := scaleFor(d)
	authLimit := middleware.NewRateLimiter(20*scale, 10*scale)
	uploadLimit := middleware.NewRateLimiter(20*scale, 10*scale)
	swipeLimit := middleware.NewRateLimiter(300*scale, 100*scale)
	sendLimit := middleware.NewRateLimiter(60*scale, 30*scale)
	reportLimit := middleware.NewRateLimiter(10*scale, 5*scale)

	r.GET("/health", platformhandlers.NewHealthHandler(d.DB))
	r.GET("/ws", hub.Handler())

	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimit)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService, cfg.MaxPhotoBytes), requireUser, uploadLimit)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser, swipeLimit)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, sendLimit)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, reportLimit)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	return r, nil
}

func scaleFor(d Dependencies) int {
	if d.RelaxLimits {
		return 1000
	}
	return 1
}
