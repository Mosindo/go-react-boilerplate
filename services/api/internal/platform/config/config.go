package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const minJWTSecretLength = 32

type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

type Config struct {
	Port           string
	Env            string
	DatabaseURL    string
	JWTSecret      string
	UploadDir      string
	AllowedOrigins []string
	SMTP           SMTP
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func Load() (Config, error) {
	smtpPort, _ := strconv.Atoi(getenv("SMTP_PORT", "587"))
	cfg := Config{
		Port:           getenv("PORT", "8080"),
		Env:            strings.ToLower(getenv("APP_ENV", "development")),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		SMTP: SMTP{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     smtpPort,
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker use postgres://app:app@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.SMTP.Host != "" && cfg.SMTP.From == "" {
		return Config{}, errors.New("SMTP_FROM is required when SMTP_HOST is set")
	}
	if cfg.IsProduction() && cfg.SMTP.Host == "" {
		return Config{}, errors.New("SMTP_HOST is required in production (password recovery sends email)")
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
