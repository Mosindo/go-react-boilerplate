package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/api/internal/app"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 30*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	application, err := app.New(cfg, pool, app.Options{Limits: app.DefaultLimits()})
	if err != nil {
		log.Fatal(err)
	}

	// A single HTTP server runs the Gin engine. Header timeouts protect
	// against slow-loris clients; no write timeout so WebSockets stay open.
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           application.Engine,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	go func() {
		log.Printf("api listening on %s (env=%s)", server.Addr, cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Print("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
