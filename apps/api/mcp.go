package main

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/modules/mcp"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/mcpauth"
	"github.com/FacileStudio/porte/oidc"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// mcpSetup is what the MCP server needs besides the services buildRouter
// builds. A nil setup leaves /mcp unmounted.
type mcpSetup struct {
	config mcp.Config
}

// newMCPSetup prepares the MCP server when MCP_RESOURCE is set and SSO is on,
// and nothing otherwise. The verifier runs Registre discovery here, so a wrong
// issuer fails at boot like the OIDC kit does. A person's first MCP call
// creates their account through the OIDC kit, as a first web login would;
// Registre decides who may get a token for this resource at all.
func newMCPSetup(ctx context.Context, appEnv env.Config, sqlDB *sql.DB, identities porte.IdentityStore, oidcKit *oidc.Kit, appLogger *slog.Logger) (*mcpSetup, error) {
	if appEnv.MCPResource == "" || !oidcKit.Enabled() {
		return nil, nil
	}
	verifier, err := mcpauth.Verifier(ctx, mcpauth.Config{
		Issuer:      appEnv.Porte.Issuer,
		Resource:    appEnv.MCPResource,
		ScopePrefix: "journal",
		Identities:  identities,
		Provision:   true,
		Accounts:    oidcKit,
		Logger:      appLogger,
	})
	if err != nil {
		return nil, err
	}
	appLogger.Info("mcp server on", slog.String("resource", appEnv.MCPResource))
	return &mcpSetup{config: mcp.Config{
		Resource: appEnv.MCPResource,
		Issuer:   appEnv.Porte.Issuer,
		Version:  version,
		Origins:  appEnv.CORSAllowedOrigins,
		StateKey: appEnv.MCPStateKey(),
		Verifier: verifier,
		Pending:  mcpkit.PostgresPending{DB: sqlDB},
		Logger:   appLogger,
	}}, nil
}

// service builds the MCP server over deps, or nothing for a nil setup.
func (s *mcpSetup) service(deps mcp.Deps) (*mcp.Service, error) {
	if s == nil {
		return nil, nil
	}
	return mcp.NewService(deps, s.config)
}
