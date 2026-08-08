package schemas_test

import (
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/schemas"
)

// Adopting porte moves sessions from Journal's own table to porte's. Both
// store the SHA-256 hex of a 32 byte token and nothing else, so the rows can
// be carried across and the credential already in somebody's browser keeps
// working. If this test fails, the deploy signs every user out.
func TestAdoptingPorteDoesNotSignAnybodyOut(t *testing.T) {
	db := testdb.Open(t)

	// users is created by AutoMigrate rather than by hand, because the
	// fidelity of this fixture is the whole point: GORM names the unique
	// index behind `uniqueIndex` itself, and a hand-written UNIQUE produces
	// a table the second AutoMigrate then tries to fix and fails on.
	if err := db.AutoMigrate(&schemas.User{}); err != nil {
		t.Fatalf("seed the pre-porte users table: %v", err)
	}

	legacy := []string{
		`CREATE TABLE sessions (
			token text PRIMARY KEY,
			user_id bigint NOT NULL,
			expires_at timestamptz,
			created_at timestamptz
		)`,
		`INSERT INTO users (id, email, name, password_hash, is_admin, created_at)
		 VALUES (1, 'someone@facile.studio', 'Someone', 'argon2id-hash', true, now())`,
		`INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES
			('live', 1, now() + interval '10 days', now() - interval '40 days'),
			('dead', 1, now() - interval '1 day',  now() - interval '31 days')`,
	}
	for _, statement := range legacy {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("seed the pre-porte schema: %v", err)
		}
	}

	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var carried struct {
		UserID     int64
		LastUsedAt time.Time
	}
	err := db.Raw(`SELECT user_id, last_used_at FROM porte_sessions WHERE token_hash = 'live'`).Scan(&carried).Error
	if err != nil {
		t.Fatalf("read the carried session: %v", err)
	}
	if carried.UserID != 1 {
		t.Fatal("the live session did not survive the migration")
	}
	// Carrying created_at over would have put this session 40 days into the
	// idle window porte retires at 7, logging the user out on the deploy
	// that was meant to keep them.
	if time.Since(carried.LastUsedAt) > time.Hour {
		t.Fatalf("last_used_at was carried over instead of stamped: %v", carried.LastUsedAt)
	}

	var expired int64
	if err := db.Raw(`SELECT count(*) FROM porte_sessions WHERE token_hash = 'dead'`).Scan(&expired).Error; err != nil {
		t.Fatalf("count expired: %v", err)
	}
	if expired != 0 {
		t.Fatal("an already-expired session was carried over")
	}

	var legacyTable *string
	if err := db.Raw(`SELECT to_regclass('sessions')::text`).Scan(&legacyTable).Error; err != nil {
		t.Fatalf("check the legacy table: %v", err)
	}
	if legacyTable != nil {
		t.Fatal("the legacy sessions table survived the migration")
	}

	// Migrate runs on every boot, including the replica that comes up
	// second, so it has to be a no-op the second time.
	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migrate is not idempotent: %v", err)
	}
}

// porte's tables key on Journal's users, not on porte_users, so deleting an
// account has to take its sessions and its identity link with it.
func TestDeletingAUserRevokesItsSessions(t *testing.T) {
	db := testdb.Open(t)
	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	seed := []string{
		`INSERT INTO users (id, email, name, password_hash, is_admin, created_at)
		 VALUES (1, 'someone@facile.studio', 'Someone', '', false, now())`,
		`INSERT INTO porte_sessions (token_hash, user_id, expires_at) VALUES ('t', 1, now() + interval '1 day')`,
		`INSERT INTO porte_identities (user_id, provider, subject) VALUES (1, 'https://sso.test/', 'abc')`,
	}
	for _, statement := range seed {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	if err := db.Exec(`DELETE FROM users WHERE id = 1`).Error; err != nil {
		t.Fatalf("delete the user: %v", err)
	}

	for _, table := range []string{"porte_sessions", "porte_identities"} {
		var remaining int64
		if err := db.Raw(`SELECT count(*) FROM ` + table).Scan(&remaining).Error; err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if remaining != 0 {
			t.Fatalf("%s survived the user it belongs to", table)
		}
	}
}
