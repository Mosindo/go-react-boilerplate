package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 30*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	router, err := server.New(server.Deps{
		Pool:           pool,
		JWTSecret:      []byte(cfg.JWTSecret),
		AllowedOrigins: cfg.AllowedOrigins,
		TrustedProxies: cfg.TrustedProxies,
		Store:          store,
		Mailer:         chooseMailer(cfg),
		Hub:            realtime.NewHub(),
	})
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("api listening on %s (env=%s)", srv.Addr, cfg.Environment)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// chooseMailer picks how recovery codes are delivered: real SMTP when
// configured, a log-only mailer in development, nothing in production.
func chooseMailer(cfg config.Config) auth.Mailer {
	smtp := auth.SMTPConfig{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
	}
	switch {
	case smtp.Enabled():
		return auth.NewSMTPMailer(smtp)
	case cfg.IsProduction():
		log.Print("SMTP is not configured: password recovery emails will not be sent")
		return auth.DisabledMailer{}
	default:
		return auth.LogMailer{}
	}
}
