package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const minJWTSecretLength = 32

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

// Config holds every runtime setting of the API. All values come from the
// environment; nothing secret is hardcoded.
type Config struct {
	Env            string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AllowedOrigins []string
	TrustedProxies []string
	UploadDir      string

	// Optional SMTP settings used for password recovery emails.
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
}

func (c Config) IsProduction() bool {
	return c.Env == EnvProduction
}

func (c Config) SMTPEnabled() bool {
	return c.SMTPHost != "" && c.SMTPFrom != ""
}

func Load() (Config, error) {
	cfg := Config{
		Env:            strings.ToLower(getenv("APP_ENV", EnvDevelopment)),
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		SMTPHost:       strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPUsername:   strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:   os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:       strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}

	if cfg.Env != EnvDevelopment && cfg.Env != EnvProduction {
		return Config{}, fmt.Errorf("APP_ENV must be %q or %q", EnvDevelopment, EnvProduction)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker use postgres://app:app@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}

	smtpPort, err := strconv.Atoi(getenv("SMTP_PORT", "587"))
	if err != nil || smtpPort <= 0 || smtpPort > 65535 {
		return Config{}, errors.New("SMTP_PORT must be a valid port number")
	}
	cfg.SMTPPort = smtpPort

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
