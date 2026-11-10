package schemas_test

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/antenne-client/go/outbox"

	"gorm.io/gorm"
)

// legacyOutbox is the antenne_outbox row Journal declared before
// antenne-client v1, kept here only to rebuild that table exactly as
// AutoMigrate created it in production.
type legacyOutbox struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Channel   string    `gorm:"column:channel"`
	Payload   string    `gorm:"column:payload"`
	Attempts  int       `gorm:"column:attempts;not null;default:0"`
	LastError string    `gorm:"column:last_error"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (legacyOutbox) TableName() string { return "antenne_outbox" }

// describeOutbox renders the columns and indexes of antenne_outbox in the
// connection's schema, with the schema name taken out so two schemas compare.
func describeOutbox(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var schema string
	var columns, indexes []string
	queries := []struct {
		target *[]string
		sql    string
	}{
		{&columns, `SELECT column_name || ' ' || data_type || ' ' || is_nullable || ' ' || coalesce(column_default, '')
			FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'antenne_outbox'`},
		{&indexes, `SELECT indexdef FROM pg_indexes WHERE schemaname = current_schema() AND tablename = 'antenne_outbox'`},
	}
	if err := db.Raw(`SELECT current_schema()`).Scan(&schema).Error; err != nil {
		t.Fatalf("current schema: %v", err)
	}
	for _, query := range queries {
		if err := db.Raw(query.sql).Scan(query.target).Error; err != nil {
			t.Fatalf("describe the outbox: %v", err)
		}
	}
	out := append(columns, indexes...)
	for i := range out {
		out[i] = strings.ReplaceAll(out[i], schema+".", "")
	}
	slices.Sort(out)
	return out
}

// libraryOutbox is the table antenne-client's own goose migration builds.
func libraryOutbox(t *testing.T) []string {
	t.Helper()
	var described []string
	t.Run("library", func(t *testing.T) {
		db := testdb.Open(t)
		up := strings.SplitN(strings.SplitN(outbox.Migration, "-- +goose Down", 2)[0], "-- +goose Up", 2)[1]
		if err := db.Exec(up).Error; err != nil {
			t.Fatalf("apply the library migration: %v", err)
		}
		described = describeOutbox(t, db)
	})
	return described
}

// Migrate builds the outbox the library's migration builds, column for
// column and index for index, so the copy in migrate.go cannot drift.
func TestOutboxMatchesTheLibraryMigration(t *testing.T) {
	want := libraryOutbox(t)
	db := testdb.Migrated(t)
	if got := describeOutbox(t, db); !slices.Equal(got, want) {
		t.Fatalf("antenne_outbox =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The outbox Journal created before antenne-client v1 is converted in place,
// rows kept, into the library's table; a second boot changes nothing.
func TestTheLegacyOutboxIsConvertedNotDuplicated(t *testing.T) {
	want := libraryOutbox(t)
	db := testdb.Open(t)
	if err := db.AutoMigrate(&legacyOutbox{}); err != nil {
		t.Fatalf("build the legacy outbox: %v", err)
	}
	if err := db.Exec(`CREATE INDEX idx_antenne_outbox_created_at ON antenne_outbox (created_at);
		INSERT INTO antenne_outbox (channel, payload, created_at) VALUES
		('alert.fired', '{"app":"Journal"}', now()), ('alert.fired', 'not json', NULL)`).Error; err != nil {
		t.Fatalf("seed the legacy outbox: %v", err)
	}
	for range 2 {
		if err := schemas.Migrate(db); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
	if got := describeOutbox(t, db); !slices.Equal(got, want) {
		t.Fatalf("converted antenne_outbox =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	var payloads []string
	err := db.Raw(`SELECT payload::text || ' ' || idempotency_key FROM antenne_outbox ORDER BY id`).Scan(&payloads).Error
	if err != nil {
		t.Fatalf("read the converted rows: %v", err)
	}
	wantRows := []string{`{"app": "Journal"} journal_legacy_outbox_1`, `"not json" journal_legacy_outbox_2`}
	if !slices.Equal(payloads, wantRows) {
		t.Fatalf("converted rows = %q, want %q", payloads, wantRows)
	}
}

// Rolling back to v0.2 runs its AutoMigrate over the converted table, which
// adds attempts back and turns payload into text. The next boot of this
// version converts it again and keeps the keys the library wrote.
func TestTheOutboxSurvivesARollback(t *testing.T) {
	want := libraryOutbox(t)
	db := testdb.Migrated(t)
	if err := db.Exec(`INSERT INTO antenne_outbox (channel, payload, idempotency_key) VALUES ('alert.fired', '{"n":1}', 'k1')`).Error; err != nil {
		t.Fatalf("queue an event: %v", err)
	}
	if err := db.AutoMigrate(&legacyOutbox{}); err != nil {
		t.Fatalf("roll back: %v", err)
	}
	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migrate after the rollback: %v", err)
	}
	if got := describeOutbox(t, db); !slices.Equal(got, want) {
		t.Fatalf("antenne_outbox after a rollback =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	var key string
	if err := db.Raw(`SELECT idempotency_key FROM antenne_outbox`).Scan(&key).Error; err != nil || key != "k1" {
		t.Fatalf("idempotency key after a rollback = %q, %v; want k1", key, err)
	}
}
