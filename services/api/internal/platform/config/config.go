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
	AllowedOrigins []string
	UploadDir      string
	// TrustedProxies lists reverse-proxy CIDRs/IPs whose X-Forwarded-For header is
	// believed when resolving the client IP (rate limiting, logs). Empty means
	// the socket address is always used, so the header cannot be spoofed.
	TrustedProxies []string
	SMTP           SMTP
	// ExposeResetCodeInLogs prints password-reset codes to the server log when
	// no SMTP server is configured. Only meant for local development.
	LogResetCodes bool
}

// SMTP holds the optional outbound mail settings used for account recovery.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (s SMTP) Enabled() bool { return s.Host != "" }

func Load() (Config, error) {
	smtpPort, err := strconv.Atoi(getenv("SMTP_PORT", "587"))
	if err != nil {
		return Config{}, errors.New("SMTP_PORT must be a number")
	}
	cfg := Config{
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		SMTP: SMTP{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Port:     smtpPort,
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("MAIL_FROM")),
		},
		LogResetCodes: strings.EqualFold(os.Getenv("LOG_RESET_CODES"), "true"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.SMTP.Enabled() && cfg.SMTP.From == "" {
		return Config{}, errors.New("MAIL_FROM is required when SMTP_HOST is set")
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
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
