package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const minSecretLength = 32

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (s SMTPConfig) Enabled() bool {
	return s.Host != ""
}

type Config struct {
	AppEnv             string
	Port               string
	DatabaseURL        string
	JWTSecret          string
	MediaDir           string
	MediaSigningSecret string
	AllowedOrigins     []string
	TrustedProxies     []string
	SMTP               SMTPConfig
}

func (c Config) IsProduction() bool {
	return c.AppEnv == EnvProduction
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:             strings.ToLower(getenv("APP_ENV", EnvDevelopment)),
		Port:               getenv("PORT", "8080"),
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:          strings.TrimSpace(os.Getenv("JWT_SECRET")),
		MediaDir:           getenv("MEDIA_DIR", "./data/media"),
		MediaSigningSecret: strings.TrimSpace(os.Getenv("MEDIA_SIGNING_SECRET")),
		AllowedOrigins:     splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies:     splitCSV(os.Getenv("TRUSTED_PROXIES")),
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}

	if cfg.AppEnv != EnvDevelopment && cfg.AppEnv != EnvProduction {
		return Config{}, fmt.Errorf("APP_ENV must be %q or %q", EnvDevelopment, EnvProduction)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateSecret("JWT_SECRET", cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.MediaSigningSecret != "" {
		if err := validateSecret("MEDIA_SIGNING_SECRET", cfg.MediaSigningSecret); err != nil {
			return Config{}, err
		}
	}

	port, err := strconv.Atoi(getenv("SMTP_PORT", "587"))
	if err != nil || port <= 0 || port > 65535 {
		return Config{}, errors.New("SMTP_PORT must be a valid TCP port")
	}
	cfg.SMTP.Port = port
	if cfg.SMTP.Enabled() && cfg.SMTP.From == "" {
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

func validateSecret(name, secret string) error {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return fmt.Errorf("%s is required", name)
	}
	if len(trimmed) < minSecretLength {
		return fmt.Errorf("%s must be at least %d characters", name, minSecretLength)
	}

	lower := strings.ToLower(trimmed)
	for _, fragment := range []string{"change-me", "replace-me", "replace-with", "example", "placeholder", "dev-secret", "test-secret"} {
		if strings.Contains(lower, fragment) {
			return fmt.Errorf("%s must be a strong random secret, not a placeholder value", name)
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
