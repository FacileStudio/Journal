package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/internal/plans"
	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/Journal/apps/api/modules/apikeys"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"
)

const (
	testIssuer = "https://sso.test"
	secretWord = "zanzibar"
)

// tenants is two accounts on one instance. A is an owner of the instance
// workspace and holds logs, a saved query, an alert rule, a key and a source
// map for app sablier. B is a fresh account in its own personal workspace,
// with its own key for an app also named sablier. Their entries interleave in
// time. INGEST_ROUTES sends the instance's acme-* apps to the workspace acme.
type tenants struct {
	liveApp
	plans          *plans.Service
	aID, bID       int64
	a, b           string
	aKeyID         int64
	aKey, bKey     string
	bWS, acmeWS    int64
	aEntries       []int64
	bEntry         int64
	aQuery, aAlert int64
}

func newTenants(t *testing.T) tenants {
	t.Helper()
	db := testdb.Migrated(t)
	appEnv := env.Config{IngestToken: testIngestToken}
	oidcKit, sessions, users := testKitFor(t, appEnv, sqlHandle(t, db))
	f := tenants{liveApp: liveApp{db: db, sessions: sessions}, plans: plans.New(db, testIssuer, slog.New(slog.DiscardHandler))}
	f.aID, f.a = f.signIn(t, "a@facile.studio", true)
	f.bID, f.b = f.signIn(t, "b@example.test", false)
	must(t, db.Create(&schemas.WorkspaceMember{WorkspaceID: 1, UserID: f.aID, Role: schemas.RoleOwner}).Error)
	must(t, db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, 'sub-b')`, f.bID, testIssuer).Error)
	routes, err := workspace.NewService(db).SyncRoutes(context.Background(), []workspace.Route{{Slug: "acme", Name: "Acme", Prefixes: []string{"acme-"}}})
	must(t, err)
	f.router = built(t)(buildRouter(db, oidcKit, sessions, users, appEnv, slog.New(slog.DiscardHandler), nil, ssoRoutes{}, nil, tenancy{routes: routes, plans: f.plans}))
	f.seed(t)
	return f
}

func (f *tenants) seed(t *testing.T) {
	t.Helper()
	key, token, err := apikeys.NewService(f.db).Create(context.Background(), apikeys.NewKey{WorkspaceID: 1, App: "sablier"})
	must(t, err)
	f.aKeyID, f.aKey = key.ID, token
	f.bWS = f.personal(t, f.b)
	var created struct {
		Token string `json:"token"`
	}
	f.decode(t, f.call(t, http.MethodPost, "/api/apikeys", f.b, map[string]any{"app": "sablier"}), http.StatusCreated, &created)
	f.bKey = created.Token
	now := time.Now().UTC()
	f.ingest(t, testIngestToken, "sablier", "alpha "+secretWord, now.Add(-3*time.Minute))
	f.ingest(t, f.bKey, "sablier", "beta line", now.Add(-2*time.Minute))
	f.ingest(t, f.aKey, "sablier", "gamma "+secretWord, now.Add(-time.Minute))
	f.ingest(t, testIngestToken, "acme-web", "acme line", now.Add(-time.Minute))
	must(t, f.db.Model(&schemas.LogEntry{}).Where("workspace_id = 1").Order("id").Pluck("id", &f.aEntries).Error)
	must(t, f.db.Model(&schemas.LogEntry{}).Where("workspace_id = ?", f.bWS).Select("id").Scan(&f.bEntry).Error)
	must(t, f.db.Model(&schemas.Workspace{}).Where("slug = 'acme'").Select("id").Scan(&f.acmeWS).Error)
	var query struct {
		Query struct{ ID int64 } `json:"query"`
	}
	f.decode(t, f.call(t, http.MethodPost, "/api/queries", f.a, map[string]any{"workspace_id": 1, "name": "Sablier", "params": map[string]any{"app": "sablier"}}), http.StatusCreated, &query)
	f.aQuery = query.Query.ID
	var alert struct {
		Alert struct{ ID int64 } `json:"alert"`
	}
	f.decode(t, f.call(t, http.MethodPost, "/api/alerts", f.a, map[string]any{"name": "Burst", "saved_query_id": f.aQuery, "threshold": 1, "window_minutes": 5, "webhook_url": "https://hooks.example.test/a"}), http.StatusCreated, &alert)
	f.aAlert = alert.Alert.ID
	must(t, f.db.Create(&schemas.SourceMap{WorkspaceID: 1, App: "sablier", Release: "v1", File: "cart.js", Content: testMap, Bytes: len(testMap)}).Error)
}

const testMap = `{"version":3,"file":"cart.js","sources":["cart.ts"],"names":[],"mappings":"AAAA"}`

func (f *tenants) personal(t *testing.T, token string) int64 {
	t.Helper()
	var listed struct {
		Workspaces []workspace.Membership `json:"workspaces"`
	}
	f.decode(t, f.call(t, http.MethodGet, "/api/workspaces", token, nil), http.StatusOK, &listed)
	if len(listed.Workspaces) != 1 || !listed.Workspaces[0].Personal {
		t.Fatalf("a new account should hold exactly its personal workspace, got %+v", listed.Workspaces)
	}
	return listed.Workspaces[0].ID
}

func (f *tenants) ingest(t *testing.T, token, app, message string, at time.Time) {
	t.Helper()
	body := map[string]any{"app": app, "message": message, "ts": at.Format(time.RFC3339), "meta": map[string]any{"release": "v1", "stack": "at f (https://x.test/assets/cart.js:1:1)"}}
	if got := f.call(t, http.MethodPost, "/api/ingest", token, body); got.Code != http.StatusCreated {
		t.Fatalf("ingest %s: %d %s", app, got.Code, got.Body)
	}
}

func (f *tenants) call(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return call(t, f.router, method, path, token, body)
}

func (f *tenants) decode(t *testing.T, got *httptest.ResponseRecorder, want int, into any) {
	t.Helper()
	if got.Code != want {
		t.Fatalf("want %d, got %d %s", want, got.Code, got.Body)
	}
	if into != nil {
		must(t, json.Unmarshal(got.Body.Bytes(), into))
	}
}

// ids lists the entry ids a GET /api/logs page returns.
func (f *tenants) ids(t *testing.T, path, token string) []int64 {
	t.Helper()
	var page struct {
		Entries []struct{ ID int64 } `json:"entries"`
	}
	f.decode(t, f.call(t, http.MethodGet, path, token, nil), http.StatusOK, &page)
	out := make([]int64, 0, len(page.Entries))
	for _, entry := range page.Entries {
		out = append(out, entry.ID)
	}
	return out
}

func (f *tenants) limitB(entriesDaily, retentionDays int) {
	f.plans.Apply(directory.Snapshot{Version: 1, EntitlementsRevision: 1, People: []directory.Person{{
		Subject: "sub-b", Entitlements: map[string]json.RawMessage{
			plans.FeatureEntriesDaily:  json.RawMessage(itoa(entriesDaily)),
			plans.FeatureRetentionDays: json.RawMessage(itoa(retentionDays)),
		},
	}}})
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func overlaps(a, b []int64) bool {
	return slices.ContainsFunc(a, func(id int64) bool { return slices.Contains(b, id) })
}

func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
