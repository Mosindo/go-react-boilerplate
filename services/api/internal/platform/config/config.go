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
	Env            string
	UploadDir      string
	AllowedOrigins []string
	TrustedProxies []string
	SMTPAddr       string
	SMTPUsername   string
	SMTPPassword   string
	SMTPFrom       string
}

// IsProduction reports whether the API runs with APP_ENV=production.
func (c Config) IsProduction() bool { return c.Env == "production" }

func Load() (Config, error) {
	cfg := Config{
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		Env:            strings.ToLower(getenv("APP_ENV", "development")),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		SMTPAddr:       strings.TrimSpace(os.Getenv("SMTP_ADDR")),
		SMTPUsername:   strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:   os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:       strings.TrimSpace(os.Getenv("SMTP_FROM")),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker/CI use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.IsProduction() && cfg.SMTPAddr == "" {
		return Config{}, errors.New("SMTP_ADDR is required when APP_ENV=production (password recovery sends emails)")
	}
	if cfg.SMTPAddr != "" && cfg.SMTPFrom == "" {
		return Config{}, errors.New("SMTP_FROM is required when SMTP_ADDR is set")
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
