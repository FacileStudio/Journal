package antenne

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/gorm"
)

func newTestService(t *testing.T, enabled bool) (*Service, *gorm.DB, schemas.AlertRule) {
	t.Helper()
	orm := testdb.Migrated(t)
	sqlDB, err := orm.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	setting := schemas.AppSetting{ID: 1, AntenneURL: "https://antenne.test", AntenneSecret: "s", AntenneEnabled: enabled}
	query := schemas.SavedQuery{Name: "errors"}
	if err := orm.Create(&setting).Error; err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	if err := orm.Create(&query).Error; err != nil {
		t.Fatalf("seed query: %v", err)
	}
	rule := schemas.AlertRule{Name: "spike", SavedQueryID: query.ID, Provider: schemas.AlertProviderAntenne,
		Threshold: 1, WindowMinutes: 5, Enabled: true}
	if err := orm.Create(&rule).Error; err != nil {
		t.Fatalf("seed rule: %v", err)
	}
	return NewService(orm, sqlDB, slog.New(slog.NewTextHandler(io.Discard, nil))), orm, rule
}

func countRows(t *testing.T, db *sql.DB, query string) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

// A firing marks the rule and queues its event in one transaction, a replay
// of the same firing queues nothing more, and the drainer, with no bus,
// leaves the event queued.
func TestFireAlertQueuesTheEventWithTheFiring(t *testing.T) {
	service, orm, rule := newTestService(t, true)
	firedAt := time.Now().UTC().Truncate(time.Millisecond)
	for range 2 {
		if err := service.FireAlert(context.Background(), rule, 7, firedAt); err != nil {
			t.Fatalf("fire: %v", err)
		}
	}
	if got := countRows(t, service.db, `SELECT count(*) FROM antenne_outbox WHERE channel = 'alert.fired' AND sent_at IS NULL`); got != 1 {
		t.Fatalf("queued %d events, want 1", got)
	}
	assertFired(t, orm, rule.ID, true)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	service.Drain(ctx)
	if got := countRows(t, service.db, `SELECT count(*) FROM antenne_outbox WHERE sent_at IS NULL`); got != 1 {
		t.Fatalf("an event with no bus to take it was marked sent: %d unsent", got)
	}
}

// With publishing off the rule is still marked fired and nothing is queued.
func TestFireAlertQueuesNothingWithPublishingOff(t *testing.T) {
	service, orm, rule := newTestService(t, false)
	if err := service.FireAlert(context.Background(), rule, 7, time.Now()); err != nil {
		t.Fatalf("fire: %v", err)
	}
	if got := countRows(t, service.db, `SELECT count(*) FROM antenne_outbox`); got != 0 {
		t.Fatalf("queued %d events with publishing off", got)
	}
	assertFired(t, orm, rule.ID, true)
}

// An enqueue that fails rolls the firing back, so the next tick retries it.
func TestAFailedEnqueueLeavesTheRuleUnfired(t *testing.T) {
	service, orm, rule := newTestService(t, true)
	if err := orm.Exec(`DROP TABLE antenne_outbox`).Error; err != nil {
		t.Fatalf("drop the outbox: %v", err)
	}
	if err := service.FireAlert(context.Background(), rule, 7, time.Now()); err == nil {
		t.Fatal("a firing whose event could not be queued reported success")
	}
	assertFired(t, orm, rule.ID, false)
}

func assertFired(t *testing.T, orm *gorm.DB, ruleID int64, want bool) {
	t.Helper()
	var stored schemas.AlertRule
	if err := orm.First(&stored, ruleID).Error; err != nil {
		t.Fatalf("reload the rule: %v", err)
	}
	if got := stored.LastFiredAt != nil; got != want {
		t.Fatalf("rule fired = %v, want %v", got, want)
	}
}

// The alert.fired envelope keeps the keys Journal put on the wire before
// antenne-client v1, so Antenne's consumers see no change.
func TestAlertEventKeepsItsWireShape(t *testing.T) {
	rule := schemas.AlertRule{ID: 3, Name: "spike", SavedQueryID: 9, Threshold: 5, WindowMinutes: 15}
	firedAt := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	body, err := json.Marshal(alertEvent(rule, 42, firedAt))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"version":1,"app":"Journal","object":"unknown","action":"created","facile_id":"journal_alert_3",` +
		`"payload":{"alert":{"id":3,"name":"spike","saved_query_id":9,"threshold":5,"window_minutes":15},` +
		`"count":42,"fired_at":"2026-11-10T12:00:00Z"},"timestamp":"2026-11-10T12:00:00Z",` +
		`"idempotency_key":"journal_alert_3_1794312000000"}`
	if string(body) != want {
		t.Fatalf("alert.fired =\n%s\nwant\n%s", body, want)
	}
}
