package ingest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
)

func browserRequest(t *testing.T, body BrowserRequest, scope authcontext.IngestScope) *http.Request {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/ingest/browser", bytes.NewReader(encoded))
	request.Header.Set("User-Agent", "Mozilla/5.0 (test)")
	return request.WithContext(authcontext.WithIngestScope(request.Context(), scope))
}

// The app is the key's, full stop. A page that could name its own app could
// bury its errors under someone else's name, which is the whole reason the
// payload carries no app field.
func TestBrowserMetaIsServerStamped(t *testing.T) {
	scope := authcontext.IngestScope{App: "shop", Origin: "https://shop.example"}
	event := BrowserEvent{
		Message: "TypeError: cart is undefined",
		URL:     "https://shop.example/cart",
		Meta: map[string]any{
			"source":   "definitely-not-a-browser",
			"origin":   "https://evil.example",
			"password": "hunter2",
			"nested":   map[string]any{"token": "abc", "keep": "yes"},
		},
	}
	request := browserRequest(t, BrowserRequest{Release: "v1.2.3", Events: []BrowserEvent{event}}, scope)

	meta := browserMeta(event, BrowserRequest{Release: "v1.2.3"}, scope, request)

	if meta["source"] != "browser" {
		t.Fatalf("source = %v, want browser", meta["source"])
	}
	if meta["origin"] != "https://shop.example" {
		t.Fatalf("origin = %v, want the scope's origin", meta["origin"])
	}
	if meta["password"] != scrubbedValue {
		t.Fatalf("password = %v, want it scrubbed", meta["password"])
	}
	nested, ok := meta["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested meta = %T, want a map", meta["nested"])
	}
	if nested["token"] != scrubbedValue {
		t.Fatalf("nested token = %v, want it scrubbed", nested["token"])
	}
	if nested["keep"] != "yes" {
		t.Fatalf("nested keep = %v, want it untouched", nested["keep"])
	}
	if meta["release"] != "v1.2.3" || meta["user_agent"] != "Mozilla/5.0 (test)" {
		t.Fatalf("release/user_agent missing: %v", meta)
	}
}

// A page in a render loop can produce megabytes of context per event. The cap
// has to hold whatever the shape is, so the fallback keeps triage alive
// (origin, url, half a stack) rather than storing nothing at all.
func TestBrowserMetaIsCapped(t *testing.T) {
	scope := authcontext.IngestScope{App: "shop", Origin: "https://shop.example"}
	event := BrowserEvent{
		Message: "boom",
		URL:     "https://shop.example/cart",
		Stack:   strings.Repeat("at frame\n", 4000),
		Meta:    map[string]any{"payload": strings.Repeat("x", 100_000)},
	}

	meta := browserMeta(event, BrowserRequest{}, scope, browserRequest(t, BrowserRequest{}, scope))

	encoded, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(encoded) > maxBrowserMetaBytes {
		t.Fatalf("meta is %d bytes, want at most %d", len(encoded), maxBrowserMetaBytes)
	}
	if meta["url"] != "https://shop.example/cart" {
		t.Fatalf("url = %v, want it kept in the fallback", meta["url"])
	}
}

// The session id is what turns one error into the sequence it belongs to, so it
// comes from the batch, cannot be claimed by an event's meta, and survives the
// oversized-meta fallback — an event too big to store whole is exactly the one
// whose neighbours are worth finding. A batch carrying none must leave the key
// absent rather than null, because the index is partial on key presence.
//
// The oversized case needs the stack to get there: scrubValue caps a single
// string at maxBrowserStringBytes, so meta alone cannot reach the limit however
// much a page sends.
func TestBrowserMetaCarriesSessionID(t *testing.T) {
	scope := authcontext.IngestScope{App: "shop", Origin: "https://shop.example"}
	batch := BrowserRequest{SessionID: "0d5f6f4e-9d2a-4a9b-8d1e-7c6b5a4f3e2d"}

	event := BrowserEvent{
		Message: "boom",
		Meta:    map[string]any{"session_id": "someone-elses-session"},
	}
	meta := browserMeta(event, batch, scope, browserRequest(t, batch, scope))
	if meta["session_id"] != batch.SessionID {
		t.Fatalf("session_id = %v, want the batch's %q", meta["session_id"], batch.SessionID)
	}

	oversized := BrowserEvent{
		Message: "boom",
		URL:     "https://shop.example/cart",
		Stack:   strings.Repeat("at frame\n", 4000),
		Meta:    map[string]any{"payload": strings.Repeat("x", 100_000)},
	}
	reduced := browserMeta(oversized, batch, scope, browserRequest(t, batch, scope))
	if reduced["meta_error"] == nil {
		t.Fatalf("expected the oversized fallback, got %v", reduced)
	}
	if reduced["session_id"] != batch.SessionID {
		t.Fatalf("fallback session_id = %v, want %q", reduced["session_id"], batch.SessionID)
	}

	bare := browserMeta(event, BrowserRequest{}, scope, browserRequest(t, BrowserRequest{}, scope))
	if _, present := bare["session_id"]; present {
		t.Fatalf("session_id = %v, want the key absent entirely", bare["session_id"])
	}
}

func TestBrowserEventCap(t *testing.T) {
	handler := newHandler(nil)
	events := make([]BrowserEvent, maxBrowserEvents+1)
	for i := range events {
		events[i] = BrowserEvent{Message: "boom"}
	}

	recorder := httptest.NewRecorder()
	handler.browser(recorder, browserRequest(t, BrowserRequest{Events: events}, authcontext.IngestScope{App: "shop"}))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "20") {
		t.Fatalf("error body %q does not name the limit", recorder.Body.String())
	}
}

// Without a scope the handler has no app to attribute entries to, and must
// refuse rather than invent one — this is the failure mode if the route is
// ever mounted without its auth middleware.
func TestBrowserRefusesWithoutScope(t *testing.T) {
	handler := newHandler(nil)
	encoded, err := json.Marshal(BrowserRequest{Events: []BrowserEvent{{Message: "boom"}}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.browser(recorder, httptest.NewRequest(http.MethodPost, "/ingest/browser", bytes.NewReader(encoded)))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestSecondsUntilUTCMidnight(t *testing.T) {
	cases := []struct {
		now  time.Time
		want int
	}{
		{time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), 86400},
		{time.Date(2026, 8, 10, 23, 59, 30, 0, time.UTC), 30},
		{time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC), 43200},
	}
	for _, tc := range cases {
		if got := secondsUntilUTCMidnight(tc.now); got != tc.want {
			t.Fatalf("secondsUntilUTCMidnight(%v) = %d, want %d", tc.now, got, tc.want)
		}
	}
}

// Breadcrumbs sent by the SDK land in meta["breadcrumbs"] after scrubbing.
func TestBrowserBreadcrumbs(t *testing.T) {
	scope := authcontext.IngestScope{App: "shop", Origin: "https://shop.example"}
	batch := BrowserRequest{
		Breadcrumbs: []Breadcrumb{
			{Level: "info", Message: "user clicked checkout", Category: "ui", Timestamp: "2026-09-01T12:00:00Z"},
			{Level: "warn", Message: "deprecated api", Category: "console", Timestamp: "2026-09-01T12:00:01Z"},
		},
	}
	event := BrowserEvent{Message: "boom", Meta: map[string]any{}}
	meta := browserMeta(event, batch, scope, browserRequest(t, batch, scope))

	bc, ok := meta["breadcrumbs"].([]any)
	if !ok {
		t.Fatalf("breadcrumbs = %T %v, want []any", meta["breadcrumbs"], meta["breadcrumbs"])
	}
	if len(bc) != 2 {
		t.Fatalf("breadcrumbs length = %d, want 2", len(bc))
	}
	first := bc[0].(map[string]any)
	if first["message"] != "user clicked checkout" || first["category"] != "ui" {
		t.Fatalf("first breadcrumb = %v, want message=user clicked checkout category=ui", first)
	}
}

// The scrubbed keys list applies to breadcrumb data too.
func TestBrowserBreadcrumbDataScrubbed(t *testing.T) {
	scope := authcontext.IngestScope{App: "shop", Origin: "https://shop.example"}
	batch := BrowserRequest{
		Breadcrumbs: []Breadcrumb{
			{
				Level: "info", Message: "api call", Category: "console",
				Timestamp: "2026-09-01T12:00:00Z",
				Data:      map[string]any{"token": "abc123", "url": "/api/orders", "keep": "yes"},
			},
		},
	}
	event := BrowserEvent{Message: "boom", Meta: map[string]any{}}
	meta := browserMeta(event, batch, scope, browserRequest(t, batch, scope))

	bc := meta["breadcrumbs"].([]any)
	entry := bc[0].(map[string]any)
	data := entry["data"].(map[string]any)
	if data["token"] != scrubbedValue {
		t.Fatalf("token = %v, want scrubbed", data["token"])
	}
	if data["url"] != "/api/orders" {
		t.Fatalf("url = %v, want /api/orders", data["url"])
	}
	if data["keep"] != "yes" {
		t.Fatalf("keep = %v, want yes", data["keep"])
	}
}

// session_id in breadcrumb data is stripped to avoid cross-tab confusion.
func TestBrowserBreadcrumbSessionIDStripped(t *testing.T) {
	scope := authcontext.IngestScope{App: "shop", Origin: "https://shop.example"}
	batch := BrowserRequest{SessionID: "tab-session"}
	crumb := Breadcrumb{
		Level: "info", Message: "click", Category: "ui",
		Timestamp: "2026-09-01T12:00:00Z",
		Data:      map[string]any{"session_id": "other-tab"},
	}
	event := BrowserEvent{Message: "boom", Meta: map[string]any{}}
	meta := browserMeta(event, BrowserRequest{SessionID: "tab-session", Breadcrumbs: []Breadcrumb{crumb}}, scope, browserRequest(t, batch, scope))

	bc := meta["breadcrumbs"].([]any)
	entry := bc[0].(map[string]any)
	data := entry["data"].(map[string]any)
	if _, present := data["session_id"]; present {
		t.Fatalf("session_id = %v, want it stripped from breadcrumb data", data["session_id"])
	}
}

// More than 50 breadcrumbs are capped to 50.
func TestBrowserBreadcrumbsCapped(t *testing.T) {
	scope := authcontext.IngestScope{App: "shop", Origin: "https://shop.example"}
	raw := make([]Breadcrumb, 60)
	for i := range raw {
		raw[i] = Breadcrumb{Level: "info", Message: "crumb", Category: "console", Timestamp: "2026-09-01T12:00:00Z"}
	}
	batch := BrowserRequest{Breadcrumbs: raw}
	event := BrowserEvent{Message: "boom"}
	meta := browserMeta(event, batch, scope, browserRequest(t, batch, scope))

	bc := meta["breadcrumbs"].([]any)
	if len(bc) > maxBrowserBreadcrumbs {
		t.Fatalf("breadcrumbs = %d, want at most %d", len(bc), maxBrowserBreadcrumbs)
	}
}
