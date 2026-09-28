package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const minJWTSecretLength = 32

// Environments recognised by APP_ENV.
const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"
)

// SMTP holds outgoing mail settings. When Host is empty the log mailer is used
// (development/test only; production requires SMTP).
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (s SMTP) Configured() bool { return s.Host != "" }

type Config struct {
	AppEnv         string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AllowedOrigins []string
	// TrustedProxies lists proxy IPs/CIDRs whose X-Forwarded-For is honoured
	// when deriving the client IP (used by per-IP rate limiting). Empty means
	// the socket address is always used, so the header cannot be spoofed.
	TrustedProxies []string
	SMTP           SMTP
}

func (c Config) IsProduction() bool { return c.AppEnv == EnvProduction }

func Load() (Config, error) {
	cfg := Config{
		AppEnv:         strings.ToLower(getenv("APP_ENV", EnvDevelopment)),
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		SMTP: SMTP{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}

	switch cfg.AppEnv {
	case EnvDevelopment, EnvTest, EnvProduction:
	default:
		return Config{}, fmt.Errorf("APP_ENV must be one of development, test, production (got %q)", cfg.AppEnv)
	}

	port := 587
	if raw := strings.TrimSpace(os.Getenv("SMTP_PORT")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 65535 {
			return Config{}, errors.New("SMTP_PORT must be a valid TCP port")
		}
		port = parsed
	}
	cfg.SMTP.Port = port

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker/CI use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.IsProduction() {
		if cfg.SMTP.Host == "" || cfg.SMTP.From == "" {
			return Config{}, errors.New("SMTP_HOST and SMTP_FROM are required when APP_ENV=production")
		}
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
