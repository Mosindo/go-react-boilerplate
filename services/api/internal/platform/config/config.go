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

func (c SMTPConfig) Enabled() bool {
	return c.Host != "" && c.From != ""
}

type Config struct {
	Env                string
	Port               string
	DatabaseURL        string
	JWTSecret          string
	MediaSigningSecret string
	UploadDir          string
	AllowedOrigins     []string
	SMTP               SMTPConfig
}

func (c Config) IsProduction() bool {
	return c.Env == EnvProduction
}

func Load() (Config, error) {
	cfg := Config{
		Env:                strings.ToLower(getenv("APP_ENV", EnvProduction)),
		Port:               getenv("PORT", "8080"),
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:          strings.TrimSpace(os.Getenv("JWT_SECRET")),
		MediaSigningSecret: strings.TrimSpace(os.Getenv("MEDIA_SIGNING_SECRET")),
		UploadDir:          strings.TrimSpace(getenv("UPLOAD_DIR", "./data/uploads")),
		AllowedOrigins:     splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		SMTP: SMTPConfig{
			Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
			Username: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
		},
	}

	switch cfg.Env {
	case EnvDevelopment, EnvProduction, EnvTest:
	default:
		return Config{}, fmt.Errorf("APP_ENV must be one of %s, %s, %s", EnvDevelopment, EnvProduction, EnvTest)
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required (for Docker/CI use postgres://postgres:postgres@postgres:5432/app?sslmode=disable)")
	}
	if err := validateJWTSecret(cfg.JWTSecret); err != nil {
		return Config{}, err
	}
	if cfg.MediaSigningSecret != "" && len(cfg.MediaSigningSecret) < minJWTSecretLength {
		return Config{}, fmt.Errorf("MEDIA_SIGNING_SECRET must be at least %d characters when set", minJWTSecretLength)
	}
	if cfg.UploadDir == "" {
		return Config{}, errors.New("UPLOAD_DIR must not be empty")
	}

	smtpPort := strings.TrimSpace(getenv("SMTP_PORT", "587"))
	port, err := strconv.Atoi(smtpPort)
	if err != nil || port <= 0 || port > 65535 {
		return Config{}, errors.New("SMTP_PORT must be a valid TCP port")
	}
	cfg.SMTP.Port = port
	if cfg.SMTP.Host != "" && cfg.SMTP.From == "" {
		return Config{}, errors.New("SMTP_FROM is required when SMTP_HOST is set")
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
