package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const minJWTSecretLength = 32

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (s SMTPConfig) Enabled() bool {
	return s.Host != "" && s.From != ""
}

type Config struct {
	Env            string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AllowedOrigins []string
	TrustedProxies []string
	UploadDir      string
	SMTP           SMTPConfig
	// RateLimitDisabled is meant for automated tests only; refused in production.
	RateLimitDisabled bool
}

func (c Config) IsProduction() bool {
	return c.Env == "production"
}

func Load() (Config, error) {
	cfg := Config{
		Env:               strings.ToLower(getenv("APP_ENV", "development")),
		Port:              getenv("PORT", "8080"),
		DatabaseURL:       strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:         strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AllowedOrigins:    splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies:    splitCSV(os.Getenv("TRUSTED_PROXIES")),
		UploadDir:         getenv("UPLOAD_DIR", "./data/uploads"),
		RateLimitDisabled: strings.EqualFold(strings.TrimSpace(os.Getenv("RATE_LIMIT_DISABLED")), "true"),
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}

	if cfg.Env != "development" && cfg.Env != "production" && cfg.Env != "test" {
		return Config{}, fmt.Errorf("APP_ENV must be development, test or production (got %q)", cfg.Env)
	}
	if cfg.RateLimitDisabled && cfg.IsProduction() {
		return Config{}, errors.New("RATE_LIMIT_DISABLED cannot be used in production")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (example: postgres://app:app@localhost:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	port, err := strconv.Atoi(getenv("SMTP_PORT", "587"))
	if err != nil || port <= 0 || port > 65535 {
		return Config{}, errors.New("SMTP_PORT must be a valid TCP port")
	}
	cfg.SMTP.Port = port

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
