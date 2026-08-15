package env

import (
	"fmt"

	"github.com/FacileStudio/porte"
	troncenv "github.com/FacileStudio/tronc/env"
)

// Config is the process configuration: the tronc core plus Journal's ingest,
// registration, retention and OIDC policy.
type Config struct {
	troncenv.Core
	IngestToken         string
	AllowRegistration   bool
	RetentionDays       int
	WebhookAllowedHosts []string
	AvatarDir           string

	// Porte is the suite's OIDC contract, unchanged: OIDC_ISSUER enables
	// it and makes the other four required. porte.Config.Validate does the
	// checking, at boot, in oidc.New — a missing client secret must not
	// become a 500 on the first login attempt three days later.
	Porte porte.Config
}

// Load reads and validates the environment, wiring the OIDC config through
// porte's own Config.
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
		AvatarDir:           troncenv.String("AVATAR_DIR", "/data/avatars"),
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

	cfg.Porte = porte.Config{
		ClaimsScope:  troncenv.String("OIDC_CLAIMS_SCOPE", ""),
		Issuer:       troncenv.String("OIDC_ISSUER", ""),
		ClientID:     troncenv.String("OIDC_CLIENT_ID", ""),
		ClientSecret: troncenv.String("OIDC_CLIENT_SECRET", ""),
		RedirectURL:  troncenv.String("OIDC_REDIRECT_URL", ""),
		SuccessURL:   troncenv.String("OIDC_SUCCESS_URL", ""),
	}
	if cfg.Porte.SSOOnly, err = troncenv.Bool("SSO_ONLY", false); err != nil {
		return Config{}, err
	}
	if cfg.Porte.SSOOnly && !cfg.Porte.Enabled() {
		return Config{}, fmt.Errorf("SSO_ONLY=true with no OIDC_ISSUER leaves no way to sign in")
	}
	if err := cfg.Porte.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
