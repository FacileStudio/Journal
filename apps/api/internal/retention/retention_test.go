package retention

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
)

func TestSweepDeletesEntriesOutsideTheWindow(t *testing.T) {
	db := open(t)
	now := time.Now()
	for _, days := range []int{120, 91, 89, 1} {
		insert(t, db, now.Add(-time.Duration(days)*24*time.Hour))
	}

	r, err := sweep(context.Background(), db, Policy{Days: 90}, now, 1)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if r != (Result{Expired: 2}) {
		t.Fatalf("result = %+v, want 2 expired", r)
	}
	if n := count(t, db, `SELECT count(*) FROM log_entries WHERE created_at < $1`, now.Add(-90*24*time.Hour)); n != 0 {
		t.Fatalf("%d entries older than the window survived", n)
	}
	if n := count(t, db, `SELECT count(*) FROM log_entries`); n != 2 {
		t.Fatalf("%d entries left, want the 2 inside the window", n)
	}
}

func TestSweepKeepsTheNewestEntriesUnderTheCeiling(t *testing.T) {
	db := open(t)
	now := time.Now()
	for range 5 {
		insert(t, db, now)
	}
	newest := count(t, db, `SELECT max(id) FROM log_entries`)

	r, err := sweep(context.Background(), db, Policy{Days: 90, MaxEntries: 3}, now, 2)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if r != (Result{Overflow: 2}) {
		t.Fatalf("result = %+v, want 2 over the ceiling", r)
	}
	if n := count(t, db, `SELECT count(*) FROM log_entries WHERE id > $1`, newest-3); n != 3 {
		t.Fatalf("%d of the 3 newest entries left, want all of them", n)
	}
	if n := count(t, db, `SELECT count(*) FROM log_entries`); n != 3 {
		t.Fatalf("%d entries left, want the ceiling of 3", n)
	}
	if again, err := sweep(context.Background(), db, Policy{MaxEntries: 3}, now, 2); err != nil || again != (Result{}) {
		t.Fatalf("second sweep = %+v, %v; want nothing to delete", again, err)
	}
}

func open(t *testing.T) *sql.DB {
	t.Helper()
	db, err := testdb.Migrated(t).DB()
	if err != nil {
		t.Fatalf("database handle: %v", err)
	}
	return db
}

func insert(t *testing.T, db *sql.DB, at time.Time) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO log_entries (app, level, message, created_at, received_at)
		VALUES ('test', 'info', 'line', $1, now())`, at); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}
