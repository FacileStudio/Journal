package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL         string
	Port                string
	LogLevel            string
	IngestToken         string
	AllowRegistration   bool
	AllowedOrigins      []string
	RetentionDays       int
	WebhookAllowedHosts []string
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:       valueOrDefault("DATABASE_URL", "postgres://journal:journal@localhost:5432/journal?sslmode=disable"),
		Port:              valueOrDefault("PORT", "4010"),
		LogLevel:          valueOrDefault("LOG_LEVEL", "info"),
		IngestToken:       os.Getenv("INGEST_TOKEN"),
		AllowRegistration: boolOrDefault("ALLOW_REGISTRATION", true),
	}

	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be a valid TCP port")
	}
	if err := validateLogLevel(cfg.LogLevel); err != nil {
		return Config{}, err
	}

	retentionDays, err := strconv.Atoi(strings.TrimSpace(valueOrDefault("RETENTION_DAYS", "90")))
	if err != nil || retentionDays < 0 {
		return Config{}, fmt.Errorf("RETENTION_DAYS must be a non-negative integer")
	}
	cfg.RetentionDays = retentionDays

	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" {
		origins = os.Getenv("DOMAINS")
	}
	if origins != "" {
		cfg.AllowedOrigins = strings.Split(origins, ",")
		for i := range cfg.AllowedOrigins {
			cfg.AllowedOrigins[i] = strings.TrimSpace(cfg.AllowedOrigins[i])
		}
	} else {
		cfg.AllowedOrigins = []string{}
	}

	cfg.WebhookAllowedHosts = splitList(os.Getenv("WEBHOOK_ALLOWED_HOSTS"))

	return cfg, nil
}

func splitList(value string) []string {
	result := []string{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func valueOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func boolOrDefault(key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "":
		return fallback
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func validateLogLevel(level string) error {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error")
	}
}
