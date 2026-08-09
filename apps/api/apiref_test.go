package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/avatarfs"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// testRouter builds the real router with a nil database. Nothing here serves a
// request that would touch it: the tests walk the route tree and read the
// reference, both of which are assembled before any handler runs.
func testKitFor(t *testing.T, appEnv env.Config, db *gorm.DB, sqlDB *sql.DB) (*oidc.Kit, *session.Manager, *local.Kit, *avatarfs.Store) {
	t.Helper()
	store := portepg.New(sqlDB)
	users := auth.NewUserStore(db)
	sessions, err := session.New(appEnv.Porte, session.Deps{Sessions: store.Sessions()})
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	avatars, err := avatarfs.New(t.TempDir(), "/avatars")
	if err != nil {
		t.Fatalf("avatarfs.New: %v", err)
	}
	kit, err := oidc.New(context.Background(), appEnv.Porte, oidc.Deps{
		Users:       users,
		Identities:  store.Identities(),
		Sessions:    sessions,
		Codes:       store.LoginCodes(),
		Avatars:     avatars,
		ConfigExtra: auth.ConfigExtra(appEnv.AllowRegistration),
	})
	if err != nil {
		t.Fatalf("build the kit: %v", err)
	}
	passwords, err := local.New(local.Config{AllowRegistration: appEnv.AllowRegistration}, local.Deps{
		Users:      users,
		Identities: store.Identities(),
		Sessions:   sessions,
		Count:      users.CountUsers,
	})
	if err != nil {
		t.Fatalf("build the password kit: %v", err)
	}
	return kit, sessions, passwords, avatars
}

func testRouter(t *testing.T, appEnv env.Config) chi.Router {
	t.Helper()
	kit, sessions, passwords, avatars := testKitFor(t, appEnv, nil, nil)
	return buildRouter(nil, kit, sessions, passwords, avatars, appEnv, slog.New(slog.DiscardHandler))
}

// ssoEnv points the kit at a stub issuer, which is what it takes to see the
// OIDC routes at all: porte does not register them when OIDC_ISSUER is unset,
// so a router built without one documents nothing about the SSO half.
func ssoEnv(t *testing.T) env.Config {
	t.Helper()
	var issuer string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/token",
			"jwks_uri":                              issuer + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	}))
	t.Cleanup(provider.Close)
	issuer = provider.URL

	return env.Config{Porte: porte.Config{
		Issuer:       issuer,
		ClientID:     "journal",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost:4010/api/auth/oidc/callback",
		SuccessURL:   "http://localhost:4010/",
	}}
}

// TestEveryRouteIsDocumented is the reason the registry can be trusted: a route
// added to any module's RegisterRoutes without a matching entry fails here.
func TestEveryRouteIsDocumented(t *testing.T) {
	for name, appEnv := range map[string]env.Config{"passwords": {}, "sso": ssoEnv(t)} {
		if missing := apiref.Undocumented(testRouter(t, appEnv), referenceConfig()); len(missing) > 0 {
			t.Errorf("%s: routes missing from the API registry: %v", name, missing)
		}
	}
}

func TestReferenceIsServed(t *testing.T) {
	router := testRouter(t, env.Config{})

	page := httptest.NewRecorder()
	router.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("GET /docs = %d, want 200", page.Code)
	}

	spec := httptest.NewRecorder()
	router.ServeHTTP(spec, httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil))
	if spec.Code != http.StatusOK {
		t.Fatalf("GET /docs/openapi.json = %d, want 200", spec.Code)
	}

	body, err := io.ReadAll(spec.Body)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var document struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("spec is not JSON: %v", err)
	}
	if document.OpenAPI != "3.1.0" {
		t.Errorf("openapi = %q, want 3.1.0", document.OpenAPI)
	}
	if _, ok := document.Paths["/ingest"]["post"]; !ok {
		t.Errorf("spec does not describe POST /ingest")
	}
}
