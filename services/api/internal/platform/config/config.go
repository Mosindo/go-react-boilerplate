package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const minJWTSecretLength = 32

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AppEnv         string
	AppBaseURL     string
	AllowedOrigins []string
	UploadDir      string
	TrustedProxies []string
	SMTP           SMTPConfig
}

// SMTPConfig is optional. Without a host, outgoing e-mails (password reset) are
// written to the server log instead, which is the intended behaviour in development.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (c Config) IsProduction() bool { return c.AppEnv == "production" }

func Load() (Config, error) {
	cfg := Config{
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AppEnv:         strings.ToLower(getenv("APP_ENV", "development")),
		AppBaseURL:     strings.TrimSpace(os.Getenv("APP_BASE_URL")),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     getenv("SMTP_PORT", "587"),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker/CI use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.SMTP.Host != "" && cfg.SMTP.From == "" {
		return Config{}, errors.New("SMTP_FROM is required when SMTP_HOST is set")
	}
	if cfg.IsProduction() && cfg.SMTP.Host == "" {
		return Config{}, errors.New("SMTP_HOST is required in production so that account recovery e-mails can be delivered")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
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
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
