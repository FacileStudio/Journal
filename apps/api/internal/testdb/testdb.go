// Package testdb opens a migrated, isolated database for a test.
//
// Postgres is the only database in this suite, tests included: a test on
// SQLite builds a different schema from the same struct tags and then passes,
// proving nothing about the DDL that ships. It wraps tronc/testdb, which gives
// every call a schema of its own.
package testdb

import (
	"context"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/migrations"
	troncdb "github.com/FacileStudio/tronc/testdb"

	"gorm.io/gorm"
)

const envVar = "JOURNAL_TEST_DATABASE_URL"

func open(t *testing.T, migrate func(*gorm.DB) error) *gorm.DB {
	t.Helper()
	url, ok := troncdb.URLFrom(envVar)
	if !ok {
		t.Skip(envVar + " is unset")
	}
	return troncdb.OpenIsolated(t, url, troncdb.Config{Prefix: "journal_test", Migrate: migrate})
}

// Open returns an empty database scoped to a schema of this test's own, or
// skips when JOURNAL_TEST_DATABASE_URL is unset.
func Open(t *testing.T) *gorm.DB {
	return open(t, func(*gorm.DB) error { return nil })
}

// Migrated is Open plus the app's migrations, which is what most tests want.
func Migrated(t *testing.T) *gorm.DB {
	return open(t, Up)
}

// Up applies the migrations to db as the binary does at boot, under an
// advisory lock of the test schema's own, so test packages migrating at once
// do not queue behind the boot's single lock.
func Up(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	cfg := migrations.Config(sqlDB, nil)
	if cfg.LockID, err = troncdb.LockID(db); err != nil {
		return err
	}
	return migrations.Up(context.Background(), cfg)
}
