package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const minJWTSecretLength = 32

type Config struct {
	Env            string // development | production
	Port           string
	DatabaseURL    string
	JWTSecret      string
	UploadDir      string
	AppBaseURL     string // used to build links in emails
	AllowedOrigins []string
	TrustedProxies []string
	SMTP           SMTPConfig
	MaxPhotoBytes  int64
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (s SMTPConfig) Enabled() bool { return s.Host != "" && s.From != "" }

func (c Config) IsProduction() bool { return c.Env == "production" }

func Load() (Config, error) {
	cfg := Config{
		Env:            strings.ToLower(getenv("APP_ENV", "development")),
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		AppBaseURL:     strings.TrimRight(strings.TrimSpace(os.Getenv("APP_BASE_URL")), "/"),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     getenv("SMTP_PORT", "587"),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
		MaxPhotoBytes: 8 << 20,
	}

	if raw := strings.TrimSpace(os.Getenv("MAX_PHOTO_BYTES")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 1<<10 || n > 32<<20 {
			return Config{}, errors.New("MAX_PHOTO_BYTES must be between 1024 and 33554432")
		}
		cfg.MaxPhotoBytes = n
	}

	if cfg.Env != "development" && cfg.Env != "production" && cfg.Env != "test" {
		return Config{}, errors.New("APP_ENV must be development, production or test")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker use postgres://app:app@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.IsProduction() {
		if !cfg.SMTP.Enabled() {
			return Config{}, errors.New("SMTP_HOST and SMTP_FROM are required in production (password recovery sends emails)")
		}
		if cfg.AppBaseURL == "" {
			return Config{}, errors.New("APP_BASE_URL is required in production (links in emails)")
		}
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func validateJWTSecret(secret string) error {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return errors.New("JWT_SECRET is required")
	}
	if len(trimmed) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLength)
	}

	lower := strings.ToLower(trimmed)
	for _, fragment := range []string{"change-me", "replace-me", "replace-with", "example", "placeholder", "dev-secret", "test-secret"} {
		if strings.Contains(lower, fragment) {
			return errors.New("JWT_SECRET must be a strong random secret, not a placeholder value")
		}
	}

	return nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
