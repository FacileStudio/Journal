package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/modules/antenne"
	"github.com/FacileStudio/Journal/apps/api/schemas"
)

const storedBusSecret = "stored-bus-secret"

func newAntenneApp(t *testing.T) liveApp {
	t.Helper()
	db := testdb.Migrated(t)
	appEnv := env.Config{IngestToken: testIngestToken}
	logger := slog.New(slog.DiscardHandler)
	oidcKit, sessions, users := testKitFor(t, appEnv, sqlHandle(t, db))
	service := antenne.NewService(db, sqlHandle(t, db), logger)
	router := buildRouter(db, oidcKit, sessions, users, appEnv, logger, service, ssoRoutes{})
	stored := schemas.AppSetting{ID: 1, AntenneURL: "https://bus.example.test", AntenneSecret: storedBusSecret}
	if err := db.Create(&stored).Error; err != nil {
		t.Fatalf("seed the settings row: %v", err)
	}
	return liveApp{router: router, db: db, sessions: sessions}
}

// The bus secret is app-wide: a signed-in account without is_admin neither
// reads nor changes it.
func TestAntenneSettingsRefuseANonAdmin(t *testing.T) {
	app := newAntenneApp(t)
	_, member := app.signIn(t, "member@facile.studio", false)
	put := map[string]any{"antenne_url": "https://elsewhere.example.test", "antenne_secret": "theirs", "antenne_enabled": false}
	for method, body := range map[string]any{http.MethodGet: nil, http.MethodPut: put} {
		recorder := call(t, app.router, method, "/api/settings/antenne", member, body)
		if recorder.Code != http.StatusForbidden {
			t.Errorf("%s /api/settings/antenne as a non-admin = %d, want 403", method, recorder.Code)
		}
		if strings.Contains(recorder.Body.String(), storedBusSecret) {
			t.Errorf("%s /api/settings/antenne leaked the secret to a non-admin", method)
		}
	}
}

// An admin gets the URL and whether a secret is set, never the secret, and a
// save without antenne_secret keeps the stored one.
func TestAntenneSettingsServeAnAdminWithoutTheSecret(t *testing.T) {
	app := newAntenneApp(t)
	_, admin := app.signIn(t, "admin@facile.studio", true)

	get := call(t, app.router, http.MethodGet, "/api/settings/antenne", admin, nil)
	var settings antenne.PoolSettingsResponse
	if err := json.Unmarshal(get.Body.Bytes(), &settings); err != nil || get.Code != http.StatusOK {
		t.Fatalf("GET = %d %s", get.Code, get.Body.String())
	}
	if strings.Contains(get.Body.String(), storedBusSecret) || !settings.Settings.SecretSet || settings.Settings.URL == "" {
		t.Fatalf("GET = %s, want the URL and secret_set without the secret", get.Body.String())
	}

	put := call(t, app.router, http.MethodPut, "/api/settings/antenne", admin,
		map[string]any{"antenne_url": "https://bus.example.test", "antenne_enabled": false})
	if put.Code != http.StatusOK || strings.Contains(put.Body.String(), storedBusSecret) {
		t.Fatalf("PUT = %d %s", put.Code, put.Body.String())
	}
	var row schemas.AppSetting
	if err := app.db.First(&row, 1).Error; err != nil {
		t.Fatal(err)
	}
	if row.AntenneSecret != storedBusSecret {
		t.Fatal("a save without antenne_secret must keep the stored secret")
	}
}
