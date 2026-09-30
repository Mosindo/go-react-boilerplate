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

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (s SMTPConfig) Enabled() bool {
	return s.Host != "" && s.From != ""
}

type Config struct {
	AppEnv         string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	AllowedOrigins []string
	TrustedProxies []string
	UploadDir      string
	SMTP           SMTPConfig
	Push           PushConfig
}

// PushConfig enables native push through the Expo Push service.
type PushConfig struct {
	Enabled     bool
	ExpoURL     string
	AccessToken string
}

func (c Config) IsProduction() bool {
	return c.AppEnv == EnvProduction
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:         strings.ToLower(getenv("APP_ENV", EnvDevelopment)),
		Port:           getenv("PORT", "8080"),
		DatabaseURL:    strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:      strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
		UploadDir:      getenv("UPLOAD_DIR", "./data/uploads"),
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}

	cfg.Push = PushConfig{
		Enabled:     strings.EqualFold(getenv("PUSH_ENABLED", "false"), "true"),
		ExpoURL:     strings.TrimSpace(os.Getenv("EXPO_PUSH_URL")),
		AccessToken: strings.TrimSpace(os.Getenv("EXPO_ACCESS_TOKEN")),
	}

	switch cfg.AppEnv {
	case EnvDevelopment, EnvProduction, EnvTest:
	default:
		return Config{}, fmt.Errorf("APP_ENV must be one of development, production, test (got %q)", cfg.AppEnv)
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}

	smtpPort, err := strconv.Atoi(getenv("SMTP_PORT", "587"))
	if err != nil || smtpPort <= 0 || smtpPort > 65535 {
		return Config{}, errors.New("SMTP_PORT must be a valid TCP port")
	}
	cfg.SMTP.Port = smtpPort
	if cfg.SMTP.Host != "" && cfg.SMTP.From == "" {
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
