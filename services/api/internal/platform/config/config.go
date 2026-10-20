package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const minJWTSecretLength = 32

type Config struct {
	Env            string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AllowedOrigins []string
	UploadDir      string
	AppName        string
	TrustedProxies []string
	SMTPHost       string
	SMTPPort       string
	SMTPUsername   string
	SMTPPassword   string
	SMTPFrom       string
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func Load() (Config, error) {
	cfg := Config{
		Env:            strings.ToLower(getenv("APP_ENV", "development")),
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		AppName:        getenv("APP_NAME", "Lumen"),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		SMTPHost:       strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:       getenv("SMTP_PORT", "587"),
		SMTPUsername:   strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:   os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:       strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}

	if cfg.Env != "development" && cfg.Env != "production" && cfg.Env != "test" {
		return Config{}, errors.New("APP_ENV must be development, production or test")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker/CI use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.SMTPHost != "" && cfg.SMTPFrom == "" {
		return Config{}, errors.New("SMTP_FROM is required when SMTP_HOST is set")
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
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
