package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/modules/mcp"
	"github.com/FacileStudio/porte/mcpauth"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/FacileStudio/tronc/mcp/mcptest"
)

// TestMCPMountsBesideTheAPI builds the real router with the MCP server on and
// checks the endpoint asks for a token, its metadata is public, the confirm
// page routes need a session, and every route stays documented.
func TestMCPMountsBesideTheAPI(t *testing.T) {
	appEnv := ssoEnv(t)
	issuer := mcptest.NewIssuer(t)
	resource := "https://journal.example.test/mcp"
	verifier, err := mcpauth.Verifier(context.Background(), mcpauth.Config{
		Issuer: issuer.URL, Resource: resource, ScopePrefix: "journal", Identities: portepg.New(nil).Identities(),
	})
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}
	setup := &mcpSetup{config: mcp.Config{
		Resource: resource, Issuer: issuer.URL, Version: "test", StateKey: appEnv.MCPStateKey(), Verifier: verifier,
		Pending: mcptest.NewPending(),
	}}
	oidcKit, sessions, users := testKitFor(t, appEnv, nil)
	router := built(t)(buildRouter(nil, oidcKit, sessions, users, appEnv, slog.New(slog.DiscardHandler), nil, ssoRoutes{}, setup))

	for _, probe := range []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/mcp", http.StatusUnauthorized},
		{http.MethodGet, "/.well-known/oauth-protected-resource/mcp", http.StatusOK},
		{http.MethodGet, "/api/mcp/pending", http.StatusUnauthorized},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(probe.method, probe.path, strings.NewReader("{}")))
		if recorder.Code != probe.want {
			t.Errorf("%s %s = %d, want %d", probe.method, probe.path, recorder.Code, probe.want)
		}
	}
	if missing := apiref.Undocumented(router, referenceConfig(), "/version", "/mcp", "/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp"); len(missing) > 0 {
		t.Errorf("routes missing from the API registry: %v", missing)
	}
}
