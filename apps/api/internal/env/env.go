package env

import (
	"crypto/sha256"
	"fmt"

	"github.com/FacileStudio/porte"
	troncenv "github.com/FacileStudio/tronc/env"
)

// Config is the process configuration: the tronc core plus Journal's ingest,
// retention and OIDC policy.
type Config struct {
	troncenv.Core
	IngestToken         string
	RetentionDays       int
	WebhookAllowedHosts []string

	// Porte is the suite's OIDC contract, unchanged: OIDC_ISSUER enables
	// it and makes the other four required. porte.Config.Validate does the
	// checking, at boot, in oidc.New — a missing client secret must not
	// become a 500 on the first login attempt three days later.
	//
	// OIDC_CLI_AUDIENCE is the client id the suite's CLI holds its
	// device-grant token under, and setting it is what mounts porte's device
	// exchange so one `facile login` serves Journal too. It is deliberately
	// not OIDC_CLIENT_ID: the token being traded was minted for the CLI, not
	// for Journal. It is deliberately not the machine audience either, which
	// names this app and arms a different path.
	Porte porte.Config

	// MCPResource is the canonical URI of the remote MCP server,
	// https://journal.facile.studio/mcp in production, and the RFC 8707
	// resource Registre binds its tokens to, from MCP_RESOURCE. Empty, or
	// without OIDC_ISSUER, leaves /mcp unmounted.
	MCPResource string
}

// MCPStateKey derives the key that signs MCP confirmation state from the OIDC
// client secret, which is secret, stable across restarts and set whenever SSO
// is.
func (c Config) MCPStateKey() []byte {
	sum := sha256.Sum256([]byte("journal-mcp-state:" + c.Porte.ClientSecret))
	return sum[:]
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
		MCPResource:         troncenv.String("MCP_RESOURCE", ""),
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
		CLIAudience:  troncenv.String("OIDC_CLI_AUDIENCE", ""),
	}
	if err := cfg.Porte.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
