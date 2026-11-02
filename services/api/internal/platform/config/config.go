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
	AppBaseURL     string
	AllowedOrigins []string
	TrustedProxies []string
	SMTP           SMTPConfig
}

// SMTPConfig is optional. Without Host, password-reset links are written to
// the server log, which is only accepted outside production.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (s SMTPConfig) Enabled() bool { return s.Host != "" }

func Load() (Config, error) {
	cfg := Config{
		Env:            strings.ToLower(getenv("APP_ENV", "development")),
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
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
	}

	if cfg.Env != "development" && cfg.Env != "production" && cfg.Env != "test" {
		return Config{}, fmt.Errorf("APP_ENV must be development, production or test (got %q)", cfg.Env)
	}
	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return Config{}, fmt.Errorf("PORT must be a number (got %q)", cfg.Port)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.SMTP.Enabled() && cfg.SMTP.From == "" {
		return Config{}, errors.New("SMTP_FROM is required when SMTP_HOST is set")
	}
	if cfg.Env == "production" {
		if !cfg.SMTP.Enabled() {
			return Config{}, errors.New("SMTP_HOST is required in production so that password recovery works")
		}
		if cfg.AppBaseURL == "" {
			return Config{}, errors.New("APP_BASE_URL is required in production")
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
	for _, fragment := range []string{"change-me", "replace-me", "example", "placeholder", "dev-secret", "test-secret"} {
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
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
