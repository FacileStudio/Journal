package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/retention"
)

func (f *tenants) now() time.Time { return time.Now().UTC() }

// Over its daily cap B gets 429 and nothing is written, while the instance
// workspace, which is exempt, keeps ingesting.
func TestIsolationDailyCapRefusesOnlyTheTenant(t *testing.T) {
	f := newTenants(t)
	f.limitB(2, 7)
	f.ingest(t, f.bKey, "sablier", "second", f.now())
	before := f.entries(t, f.bWS)
	if got := f.call(t, http.MethodPost, "/api/ingest", f.bKey, map[string]any{"app": "sablier", "message": "third"}); got.Code != http.StatusTooManyRequests {
		t.Fatalf("B over its cap = %d %s, want 429", got.Code, got.Body)
	}
	if f.entries(t, f.bWS) != before {
		t.Fatal("a refused batch was written")
	}
	for range 3 {
		f.ingest(t, testIngestToken, "sablier", "instance line", f.now())
	}
}

// Without a plan value the free defaults apply, never forfait's fail-open
// unlimited: a fresh tenant is capped before any snapshot arrives.
func TestIsolationFreeDefaultsBeforeAnySnapshot(t *testing.T) {
	f := newTenants(t)
	limits, err := f.plans.Limits(context.Background(), f.bWS)
	if err != nil || limits.Exempt || limits.EntriesDaily != 10_000 || limits.RetentionDays != 7 {
		t.Fatalf("B's limits = %+v %v, want the free plan", limits, err)
	}
	instance, err := f.plans.Limits(context.Background(), 1)
	if err != nil || !instance.Exempt {
		t.Fatalf("the instance workspace must be exempt: %+v %v", instance, err)
	}
}

// B's retention removes B's old entries only; A's entries of the same age stay.
func TestIsolationRetentionIsPerWorkspace(t *testing.T) {
	f := newTenants(t)
	f.limitB(100, 1)
	old := f.now().Add(-48 * time.Hour)
	must(t, f.db.Exec(`UPDATE log_entries SET created_at = ? WHERE workspace_id IN (1, ?)`, old, f.bWS).Error)
	instance := f.entries(t, 1)
	sqlDB := sqlHandle(t, f.db)
	r, err := retention.Sweep(context.Background(), sqlDB, retention.Policy{Days: 90, Tenants: f.plans})
	if err != nil || r.Tenant != 1 {
		t.Fatalf("sweep = %+v %v, want B's one entry expired", r, err)
	}
	if f.entries(t, f.bWS) != 0 || f.entries(t, 1) != instance {
		t.Fatalf("B kept %d, A kept %d of %d", f.entries(t, f.bWS), f.entries(t, 1), instance)
	}
}

// Erasing B removes B's workspace with its keys, queries and alert rules at
// once and its log entries on the next sweep; A's counts do not move.
func TestIsolationErasureRemovesOnlyTheTenant(t *testing.T) {
	f := newTenants(t)
	instance := f.entries(t, 1)
	if got := f.call(t, http.MethodDelete, "/api/auth/me", f.b, nil); got.Code != http.StatusNoContent {
		t.Fatalf("erase B = %d %s", got.Code, got.Body)
	}
	if n := f.count(t, `SELECT count(*) FROM api_keys WHERE workspace_id = ?`, f.bWS); n != 0 {
		t.Fatalf("B's keys survived: %d", n)
	}
	r, err := retention.Sweep(context.Background(), sqlHandle(t, f.db), retention.Policy{Tenants: f.plans})
	if err != nil || r.Purged != 1 {
		t.Fatalf("sweep = %+v %v, want B's entry purged", r, err)
	}
	if f.entries(t, f.bWS) != 0 || f.entries(t, 1) != instance || f.count(t, `SELECT count(*) FROM api_keys WHERE workspace_id = 1`) != 1 {
		t.Fatal("erasure touched A's data or left B's")
	}
}
