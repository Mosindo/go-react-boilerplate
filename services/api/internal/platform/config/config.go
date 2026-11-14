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

// Enabled reports whether outgoing e-mail is configured. Without SMTP the
// API logs password-reset codes instead (development only).
func (c SMTPConfig) Enabled() bool {
	return c.Host != "" && c.From != ""
}

type Config struct {
	Env            string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	UploadDir      string
	AllowedOrigins []string
	TrustedProxies []string
	SMTP           SMTPConfig
}

func (c Config) IsProduction() bool {
	return c.Env == "production"
}

func Load() (Config, error) {
	smtpPort := 587
	if raw := strings.TrimSpace(os.Getenv("SMTP_PORT")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > 65535 {
			return Config{}, errors.New("SMTP_PORT must be a valid port number")
		}
		smtpPort = parsed
	}

	cfg := Config{
		Env:            strings.ToLower(getenv("APP_ENV", "development")),
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     smtpPort,
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}

	if cfg.Env != "development" && cfg.Env != "test" && cfg.Env != "production" {
		return Config{}, errors.New("APP_ENV must be one of development, test, production")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.IsProduction() && !cfg.SMTP.Enabled() {
		return Config{}, errors.New("SMTP_HOST and SMTP_FROM are required in production so password reset codes can be delivered")
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
