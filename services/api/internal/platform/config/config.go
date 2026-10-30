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
	EnvTest        = "test"
)

type SMTP struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func (s SMTP) Configured() bool {
	return s.Host != "" && s.From != ""
}

type Config struct {
	Env            string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AppBaseURL     string
	AllowedOrigins []string
	TrustedProxies []string
	UploadDir      string
	SMTP           SMTP
}

func (c Config) IsProduction() bool {
	return c.Env == EnvProduction
}

func Load() (Config, error) {
	smtpPort, err := parsePort(os.Getenv("SMTP_PORT"), 587)
	if err != nil {
		return Config{}, fmt.Errorf("SMTP_PORT: %w", err)
	}

	cfg := Config{
		Env:            strings.ToLower(getenv("APP_ENV", EnvDevelopment)),
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AppBaseURL:     strings.TrimSpace(os.Getenv("APP_BASE_URL")),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		SMTP: SMTP{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     smtpPort,
			User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}

	switch cfg.Env {
	case EnvDevelopment, EnvProduction, EnvTest:
	default:
		return Config{}, fmt.Errorf("APP_ENV must be one of development, production, test (got %q)", cfg.Env)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker/CI use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.IsProduction() && !cfg.SMTP.Configured() {
		return Config{}, errors.New("SMTP_HOST and SMTP_FROM are required in production so password reset emails can be delivered")
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

func parsePort(raw string, fallback int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return 0, errors.New("must be a valid port number")
	}
	return port, nil
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
