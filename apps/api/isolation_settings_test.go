package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func (f *tenants) count(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	var n int64
	must(t, f.db.Raw(query, args...).Scan(&n).Error)
	return n
}

// B lists none of A's saved queries, cannot delete one by id, and may reuse
// its name, since names are unique per workspace.
func TestIsolationSavedQueries(t *testing.T) {
	f := newTenants(t)
	if body := f.call(t, http.MethodGet, "/api/queries", f.b, nil).Body.String(); strings.Contains(body, "Sablier") {
		t.Fatalf("B listed A's saved query: %s", body)
	}
	if got := f.call(t, http.MethodDelete, fmt.Sprintf("/api/queries/%d", f.aQuery), f.b, nil); got.Code != http.StatusNotFound {
		t.Fatalf("B deleting A's query = %d, want 404", got.Code)
	}
	if f.count(t, `SELECT count(*) FROM saved_queries WHERE id = ?`, f.aQuery) != 1 {
		t.Fatal("A's saved query is gone")
	}
	f.decode(t, f.call(t, http.MethodPost, "/api/queries", f.b, map[string]any{"name": "Sablier", "params": map[string]any{}}), http.StatusCreated, nil)
}

// B lists none of A's alert rules, cannot watch A's saved query, and cannot
// pause or delete A's rule by id.
func TestIsolationAlertRules(t *testing.T) {
	f := newTenants(t)
	if body := f.call(t, http.MethodGet, "/api/alerts", f.b, nil).Body.String(); strings.Contains(body, "Burst") {
		t.Fatalf("B listed A's alert rule: %s", body)
	}
	create := map[string]any{"name": "Steal", "saved_query_id": f.aQuery, "threshold": 1, "window_minutes": 5, "webhook_url": "https://hooks.example.test/b"}
	if got := f.call(t, http.MethodPost, "/api/alerts", f.b, create); got.Code != http.StatusNotFound {
		t.Fatalf("B watching A's query = %d, want 404", got.Code)
	}
	path := fmt.Sprintf("/api/alerts/%d", f.aAlert)
	if got := f.call(t, http.MethodPatch, path, f.b, map[string]any{"enabled": false}); got.Code != http.StatusNotFound {
		t.Fatalf("B pausing A's rule = %d, want 404", got.Code)
	}
	if got := f.call(t, http.MethodDelete, path, f.b, nil); got.Code != http.StatusNotFound {
		t.Fatalf("B deleting A's rule = %d, want 404", got.Code)
	}
	if f.count(t, `SELECT count(*) FROM alert_rules WHERE id = ? AND enabled`, f.aAlert) != 1 {
		t.Fatal("A's alert rule changed")
	}
}

// The Antenne provider reaches the operators' bus, so only the instance
// workspace may use it.
func TestIsolationAntenneProviderIsInstanceOnly(t *testing.T) {
	f := newTenants(t)
	var query struct {
		Query struct{ ID int64 } `json:"query"`
	}
	f.decode(t, f.call(t, http.MethodPost, "/api/queries", f.b, map[string]any{"name": "Mine", "params": map[string]any{}}), http.StatusCreated, &query)
	alert := map[string]any{"name": "Bus", "saved_query_id": query.Query.ID, "provider": "antenne", "threshold": 1, "window_minutes": 5}
	if got := f.call(t, http.MethodPost, "/api/alerts", f.b, alert); got.Code != http.StatusBadRequest {
		t.Fatalf("B choosing antenne = %d %s, want 400", got.Code, got.Body)
	}
}

// B lists none of A's keys and cannot revoke one by id; A's key keeps working.
func TestIsolationAPIKeys(t *testing.T) {
	f := newTenants(t)
	if body := f.call(t, http.MethodGet, "/api/apikeys", f.b, nil).Body.String(); strings.Count(body, `"id"`) != 1 {
		t.Fatalf("B should list exactly its own key: %s", body)
	}
	if got := f.call(t, http.MethodDelete, fmt.Sprintf("/api/apikeys/%d", f.aKeyID), f.b, nil); got.Code != http.StatusNotFound {
		t.Fatalf("B revoking A's key = %d, want 404", got.Code)
	}
	f.ingest(t, f.aKey, "sablier", "still works", f.now())
}

// A key writes to its own workspace whatever the body says, and the legacy
// token writes to the instance workspace or a routed one, never to B's.
func TestIsolationIngestLandsInTheCredentialsWorkspace(t *testing.T) {
	f := newTenants(t)
	body := map[string]any{"app": "sablier", "message": "smuggle", "workspace_id": 1}
	if got := f.call(t, http.MethodPost, "/api/ingest", f.bKey, body); got.Code != http.StatusCreated && got.Code != http.StatusBadRequest {
		t.Fatalf("smuggled ingest = %d", got.Code)
	}
	if n := f.count(t, `SELECT count(*) FROM log_entries WHERE message = 'smuggle' AND workspace_id <> ?`, f.bWS); n != 0 {
		t.Fatalf("B's key wrote %d entries outside its workspace", n)
	}
	if n := f.count(t, `SELECT count(*) FROM log_entries WHERE workspace_id = ?`, f.bWS); n > 2 {
		t.Fatalf("B's workspace holds %d entries", n)
	}
	if n := f.count(t, `SELECT count(*) FROM log_entries WHERE workspace_id NOT IN (1, ?, ?)`, f.bWS, f.acmeWS); n != 0 {
		t.Fatalf("%d entries landed in an unexpected workspace", n)
	}
	if n := f.count(t, `SELECT count(*) FROM log_entries WHERE app = 'acme-web' AND workspace_id = ?`, f.acmeWS); n != 1 {
		t.Fatalf("the routed app did not land in its workspace: %d", n)
	}
}

// B's key sees none of A's source maps for the same app name, and resolving
// B's stack never uses A's map.
func TestIsolationSourceMaps(t *testing.T) {
	f := newTenants(t)
	if body := f.call(t, http.MethodGet, "/api/sourcemaps?release=v1", f.bKey, nil).Body.String(); strings.Contains(body, "cart.js") {
		t.Fatalf("B's key listed A's map: %s", body)
	}
	if body := f.call(t, http.MethodGet, "/api/sourcemaps/releases", f.b, nil).Body.String(); strings.Contains(body, "v1") {
		t.Fatalf("B listed A's releases: %s", body)
	}
	if body := f.call(t, http.MethodGet, fmt.Sprintf("/api/logs/%d/stack", f.aEntries[0]), f.a, nil).Body.String(); !strings.Contains(body, `"resolved":1`) {
		t.Fatalf("A's own stack should resolve through A's map: %s", body)
	}
	if body := f.call(t, http.MethodGet, fmt.Sprintf("/api/logs/%d/stack", f.bEntry), f.b, nil).Body.String(); !strings.Contains(body, `"resolved":0`) {
		t.Fatalf("B's stack resolved through A's map: %s", body)
	}
	if got := f.call(t, http.MethodDelete, "/api/sourcemaps?app=sablier&release=v1&workspace_id=1", f.b, nil); got.Code != http.StatusNotFound {
		t.Fatalf("B deleting A's maps = %d, want 404", got.Code)
	}
	if f.count(t, `SELECT count(*) FROM source_maps WHERE workspace_id = 1`) != 1 {
		t.Fatal("A's source map is gone")
	}
}

func (f *tenants) entries(t *testing.T, workspaceID int64) int64 {
	t.Helper()
	return f.count(t, `SELECT count(*) FROM log_entries WHERE workspace_id = ?`, workspaceID)
}
