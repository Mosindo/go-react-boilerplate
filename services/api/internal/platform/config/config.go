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
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AppEnv         string
	UploadDir      string
	AllowedOrigins []string
	// TrustedProxies lists reverse-proxy IPs/CIDRs whose X-Forwarded-For is trusted (client IP for rate limiting).
	TrustedProxies []string
	SMTP           SMTPConfig
}

type SMTPConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

func (s SMTPConfig) Enabled() bool { return s.Host != "" }

func Load() (Config, error) {
	cfg := Config{
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AppEnv:         getenv("APP_ENV", "development"),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		SMTP: SMTPConfig{
			Host: strings.TrimSpace(os.Getenv("SMTP_HOST")),
			User: strings.TrimSpace(os.Getenv("SMTP_USER")),
			Pass: os.Getenv("SMTP_PASS"),
			From: strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}
	cfg.SMTP.Port = 587
	if raw := strings.TrimSpace(os.Getenv("SMTP_PORT")); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port <= 0 || port > 65535 {
			return Config{}, errors.New("SMTP_PORT must be a valid port number")
		}
		cfg.SMTP.Port = port
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
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
	return cfg, nil
}

func (c Config) IsProduction() bool { return strings.EqualFold(c.AppEnv, "production") }

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
