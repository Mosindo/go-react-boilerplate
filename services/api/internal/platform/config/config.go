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
	AllowedOrigins []string
	TrustedProxies []string
	UploadDir      string
	// PasswordResetURL is the link template mailed to users; "{token}" is replaced by the reset token.
	PasswordResetURL string
	SMTP             SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// Enabled reports whether outgoing email is configured.
func (s SMTPConfig) Enabled() bool { return s.Host != "" }

// IsProduction reports whether the API runs with production safeguards.
func (c Config) IsProduction() bool { return c.AppEnv == "production" }

func Load() (Config, error) {
	cfg := Config{
		Port:             getenv("PORT", "8080"),
		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:        strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AppEnv:           strings.ToLower(getenv("APP_ENV", "development")),
		AllowedOrigins:   splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies:   splitCSV(os.Getenv("TRUSTED_PROXIES")),
		UploadDir:        getenv("UPLOAD_DIR", "./data/uploads"),
		PasswordResetURL: getenv("PASSWORD_RESET_URL", "http://localhost:8081/reset-password?token={token}"),
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     getenv("SMTP_PORT", "587"),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}

	switch cfg.AppEnv {
	case "development", "test", "production":
	default:
		return Config{}, errors.New("APP_ENV must be one of development, test, production")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker/CI use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.SMTP.Enabled() && cfg.SMTP.From == "" {
		return Config{}, errors.New("SMTP_FROM is required when SMTP_HOST is set")
	}
	if cfg.IsProduction() && !cfg.SMTP.Enabled() {
		return Config{}, errors.New("SMTP_HOST is required in production so password recovery emails can be delivered")
	}
	if !strings.Contains(cfg.PasswordResetURL, "{token}") {
		return Config{}, errors.New("PASSWORD_RESET_URL must contain the {token} placeholder")
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
