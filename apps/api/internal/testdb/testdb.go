// Package testdb opens a migrated, isolated database for a test.
//
// Postgres is the only database in this suite, tests included: a test on
// SQLite builds a different schema from the same struct tags and then passes,
// proving nothing about the DDL that ships — and it cannot run what does ship,
// which here is a DO block, to_regclass and an ON CONFLICT.
package testdb

import (
	"os"
	"strings"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open returns a migrated database scoped to a schema of this test's own, or
// skips when JOURNAL_TEST_DATABASE_URL is unset.
//
// Every caller gets its own PostgreSQL schema rather than a reset of public.
// go test runs one binary per package concurrently, so two of them dropping
// and recreating the same schema race in a way that surfaces as a nonsense
// error a long way from the cause — a missing log_entries table, a duplicate
// key in pg_type — and blames whichever test happened to lose. The search path
// is put in the connection string rather than a SET, because GORM hands out
// pooled connections and a SET reaches exactly the one it ran on, so half the
// test would silently address public instead.
func Open(t *testing.T) *gorm.DB {
	t.Helper()
	url := os.Getenv("JOURNAL_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("JOURNAL_TEST_DATABASE_URL is unset")
	}

	name := schemaName(t.Name())
	admin, err := gorm.Open(postgres.Open(url), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := admin.Exec(`DROP SCHEMA IF EXISTS ` + name + ` CASCADE`).Error; err != nil {
		t.Fatalf("drop the test schema: %v", err)
	}
	if err := admin.Exec(`CREATE SCHEMA ` + name).Error; err != nil {
		t.Fatalf("create the test schema: %v", err)
	}

	db, err := gorm.Open(postgres.Open(withSearchPath(url, name)), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open with the search path: %v", err)
	}
	t.Cleanup(func() {
		if handle, err := db.DB(); err == nil {
			_ = handle.Close()
		}
		_ = admin.Exec(`DROP SCHEMA IF EXISTS ` + name + ` CASCADE`).Error
		if handle, err := admin.DB(); err == nil {
			_ = handle.Close()
		}
	})
	return db
}

// Migrated is Open plus the app's migrations, which is what most tests want.
func Migrated(t *testing.T) *gorm.DB {
	t.Helper()
	db := Open(t)
	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// withSearchPath points a connection string at one schema.
func withSearchPath(url, schema string) string {
	separator := "?"
	if strings.Contains(url, "?") {
		separator = "&"
	}
	return url + separator + "search_path=" + schema
}

// schemaName turns a test name into a legal, unquoted identifier.
func schemaName(test string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return '_'
		}
	}, test)
	if len(safe) > 40 {
		safe = safe[:40]
	}
	return "test_" + safe
}
