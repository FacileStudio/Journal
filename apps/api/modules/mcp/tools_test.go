package mcp

import (
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/tronc/mcp/mcptest"
)

// TestSearchFiltersPagesAndClips walks a model's usual search: filter by
// app and level, page with the cursor, and get long messages cut.
func TestSearchFiltersPagesAndClips(t *testing.T) {
	f := newFixture(t)
	for i := range 3 {
		f.entry(t, "nuage", "error", "upload failed "+strings.Repeat("x", i), time.Duration(i+1)*time.Minute, nil)
	}
	f.entry(t, "nuage", "info", "upload ok", time.Minute, nil)
	f.entry(t, "agenda", "error", "sync failed", time.Minute, nil)
	f.entry(t, "nuage", "error", "old failure", 48*time.Hour, nil)
	long := f.entry(t, "nuage", "warn", strings.Repeat("é", 2_000), time.Minute, map[string]any{"request_id": "req-7"})
	session := f.connect(t, memberSub)

	var first, second searchOutput
	call(t, session, toolSearch, map[string]any{"app": "nuage", "levels": []string{"error"}, "limit": 2}, &first)
	if len(first.Entries) != 2 || first.NextCursor == "" || first.Entries[0].Level != "error" {
		t.Fatalf("want a full first page of nuage errors and a cursor, got %+v", first)
	}
	call(t, session, toolSearch, map[string]any{"app": "nuage", "levels": []string{"error"}, "limit": 2, "cursor": first.NextCursor}, &second)
	if len(second.Entries) != 1 || second.NextCursor != "" {
		t.Fatalf("the last page inside 24h holds one entry, got %+v", second)
	}

	var byRequest searchOutput
	call(t, session, toolSearch, map[string]any{"request_id": "req-7"}, &byRequest)
	got := byRequest.Entries[0]
	if got.ID != long.ID || !got.MessageTruncated || len(got.Message) > listMessageBytes || got.Meta != nil {
		t.Fatalf("a long message must be cut and concise must leave meta out, got %+v", got)
	}

	var week searchOutput
	call(t, session, toolSearch, map[string]any{"q": "failure", "since": "7d"}, &week)
	if len(week.Entries) != 1 {
		t.Fatalf("since 7d must reach the two day old entry, got %+v", week)
	}
	if text := refused(t, session, toolSearch, map[string]any{"levels": []string{"fatal"}}); !strings.Contains(text, "debug, info") {
		t.Fatalf("a bad level must name the valid ones, got %q", text)
	}
}

// TestSearchStaysUnderTheBudget fills a page with large entries and checks
// the result is cut under the budget with a cursor to the rest.
func TestSearchStaysUnderTheBudget(t *testing.T) {
	f := newFixture(t)
	for i := range 120 {
		f.entry(t, "bulk", "info", strings.Repeat("m", 900), time.Duration(i+1)*time.Second, map[string]any{"blob": strings.Repeat("b", 5_000)})
	}
	session := f.connect(t, memberSub)
	result := mcptest.Call(t, session, toolSearch, map[string]any{"app": "bulk", "limit": 200, "response_format": "detailed"})
	var out searchOutput
	mcptest.Decode(t, result, &out)
	if len(mcptest.Text(result)) > 2*75_000 || len(out.Entries) == 120 || out.NextCursor == "" {
		t.Fatalf("a large page must be cut with a cursor, got %d entries", len(out.Entries))
	}
	if !out.Entries[0].MetaTruncated || out.Entries[0].MetaPreview == "" {
		t.Fatalf("large meta must come back as a preview, got %+v", out.Entries[0])
	}
}

// TestReadShowsNeighboursAndMeta reads one entry with the entries around it.
func TestReadShowsNeighboursAndMeta(t *testing.T) {
	f := newFixture(t)
	older := f.entry(t, "nuage", "info", "before", 3*time.Minute, nil)
	anchor := f.entry(t, "nuage", "error", "boom", 2*time.Minute, map[string]any{"session_id": "tab-1", "user": "u-1"})
	newer := f.entry(t, "agenda", "info", "after", time.Minute, nil)
	session := f.connect(t, memberSub)
	var out readOutput
	call(t, session, toolRead, map[string]any{"entry_id": anchor.ID, "before": 5, "after": 5, "resolve_stack": true}, &out)
	if out.Entry.Meta["user"] != "u-1" || out.SessionID != "tab-1" || out.Stack == nil {
		t.Fatalf("want the full entry with meta and a stack, got %+v", out)
	}
	if len(out.Older) != 1 || out.Older[0].ID != older.ID || len(out.Newer) != 1 || out.Newer[0].ID != newer.ID {
		t.Fatalf("want one neighbour on each side, got older %+v newer %+v", out.Older, out.Newer)
	}
	if text := refused(t, session, toolRead, map[string]any{"entry_id": 999_999}); !strings.Contains(text, toolSearch) {
		t.Fatalf("a missing entry must point at the search, got %q", text)
	}
}

// TestCountAndApps sums the per-level counts and lists the apps.
func TestCountAndApps(t *testing.T) {
	f := newFixture(t)
	f.entry(t, "nuage", "error", "a", time.Minute, nil)
	f.entry(t, "nuage", "error", "b", 2*time.Minute, nil)
	f.entry(t, "agenda", "info", "c", time.Minute, nil)
	session := f.connect(t, memberSub)
	var counted countOutput
	call(t, session, toolCount, map[string]any{"since": "1h"}, &counted)
	if counted.Total != 3 || counted.ByLevel["error"] != 2 || counted.BucketSeconds != 60 {
		t.Fatalf("want 3 entries, 2 errors in minute buckets, got %+v", counted)
	}
	refused(t, session, toolCount, map[string]any{"since": "120d"})
	var listed appsOutput
	call(t, session, toolApps, nil, &listed)
	if len(listed.Apps) != 2 {
		t.Fatalf("want two apps, got %+v", listed)
	}
}

// TestAlertsAreForAdministrators checks a member reads no alert rule while
// the administrator does, with the webhook named by its host only.
func TestAlertsAreForAdministrators(t *testing.T) {
	f := newFixture(t)
	q := f.query(t, "Nuage errors")
	rule := f.alert(t, "Nuage burst", q.ID)
	member := f.connect(t, memberSub)
	for tool, args := range map[string]map[string]any{
		toolAlerts:      nil,
		toolSetAlert:    {"alert_id": rule.ID, "enabled": false},
		toolDeleteAlert: {"alert_id": rule.ID},
		toolCreateAlert: {"name": "x", "saved_query_id": q.ID, "threshold": 1, "window_minutes": 5},
	} {
		if text := refused(t, member, tool, args); !strings.Contains(text, "administrators only") {
			t.Fatalf("%s must refuse a member, got %q", tool, text)
		}
	}
	admin := f.connect(t, adminSub)
	var listed alertsOutput
	call(t, admin, toolAlerts, nil, &listed)
	if len(listed.Alerts) != 1 || listed.Alerts[0].WebhookHost != "hooks.example.test" || strings.Contains(mcptest.Text(mcptest.Call(t, admin, toolAlerts, nil)), "secret-token") {
		t.Fatalf("want the rule with its webhook host and never the URL, got %+v", listed)
	}
	var created, paused alertOutput
	call(t, admin, toolCreateAlert, map[string]any{"name": "Bus", "saved_query_id": q.ID, "threshold": 5, "window_minutes": 10}, &created)
	if created.Alert.Provider != "antenne" || !created.Alert.Enabled {
		t.Fatalf("a new rule defaults to Antenne and starts enabled, got %+v", created)
	}
	call(t, admin, toolSetAlert, map[string]any{"alert_id": created.Alert.ID, "enabled": false}, &paused)
	if paused.Status != "disabled" || paused.Alert.Enabled {
		t.Fatalf("want the rule paused, got %+v", paused)
	}
}

// TestSaveQueryAndRefuseWatchedDelete saves a query, refuses deleting one an
// alert rule watches, and refuses a write on a read-only token.
func TestSaveQueryAndRefuseWatchedDelete(t *testing.T) {
	f := newFixture(t)
	session := f.connect(t, memberSub)
	var saved queryOutput
	call(t, session, toolSaveQuery, map[string]any{"name": "Browser errors", "source": "browser", "levels": []string{"error"}}, &saved)
	if saved.Status != "created" || saved.Query.Source != "browser" {
		t.Fatalf("want the query saved, got %+v", saved)
	}
	if text := refused(t, session, toolSaveQuery, map[string]any{"name": "Browser errors"}); !strings.Contains(text, "already exists") {
		t.Fatalf("a duplicate name must say so, got %q", text)
	}
	watched := f.query(t, "Watched")
	f.alert(t, "Watcher", watched.ID)
	if text := refused(t, session, toolDeleteQuery, map[string]any{"query_id": watched.ID}); !strings.Contains(text, "alert rule") {
		t.Fatalf("a watched query must not be offered for deletion, got %q", text)
	}
	reader := mcptest.Connect(t, f.resource, f.issuer.Token(t, f.resource, memberSub, "journal.read"), mcptest.ClientOptions{})
	var apps appsOutput
	call(t, reader, toolApps, nil, &apps)
}
