package env

import (
	"fmt"

	troncenv "github.com/FacileStudio/tronc/env"
)

type Config struct {
	troncenv.Core
	IngestToken         string
	AllowRegistration   bool
	RetentionDays       int
	WebhookAllowedHosts []string
}

func Load() (Config, error) {
	core, err := troncenv.LoadCore()
	if err != nil {
		return Config{}, err
	}
	if core.Port < 1 || core.Port > 65535 {
		return Config{}, fmt.Errorf("PORT must be a valid TCP port")
	}

	cfg := Config{
		Core:                core,
		IngestToken:         troncenv.String("INGEST_TOKEN", ""),
		WebhookAllowedHosts: troncenv.List("WEBHOOK_ALLOWED_HOSTS"),
	}

	if cfg.AllowRegistration, err = troncenv.Bool("ALLOW_REGISTRATION", true); err != nil {
		return Config{}, err
	}

	if cfg.RetentionDays, err = troncenv.Int("RETENTION_DAYS", 90); err != nil {
		return Config{}, err
	}
	if cfg.RetentionDays < 0 {
		return Config{}, fmt.Errorf("RETENTION_DAYS must be a non-negative integer")
	}

	return cfg, nil
}
