package schemas_test

import (
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte/local"

	"gorm.io/gorm"
)

// Adopting porte moves sessions from Journal's own table to porte's. Both
// store the SHA-256 hex of a 32 byte token and nothing else, so the rows can
// be carried across and the credential already in somebody's browser keeps
// working. If this test fails, the deploy signs every user out.
// TestAdoptingPorteDoesNotSignAnybodyOut checks a pre-Porte session today is
// still a session tomorrow. users is created by AutoMigrate rather than by
// hand because the fidelity of this fixture is the whole point: GORM names the
// unique index behind uniqueIndex itself, and a hand-written UNIQUE produces
// the same constraint under a different name and the test would not actually
// be exercising the shipped migration. Carrying created_at over would put this
// session 40 days into the idle window porte retires at 7, so last_used_at is
// re-stamped rather than copied.
func TestAdoptingPorteDoesNotSignAnybodyOut(t *testing.T) {
	db := testdb.Open(t)

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

// Adopting porte/local moves the password from users.password_hash to an
// identity row. If this fails, the deploy answers "invalid email or password"
// to every correct password an existing user types — the hash is still in the
// users table and nothing reads it there any more.
//
// The row is keyed on the account id, which is what porte.LocalSubject returns
// and what porte/local looks a credential up by. Keying it on the address, as
// v0.2 did, puts the credential somewhere nothing reads.
func TestExistingPasswordsKeepWorking(t *testing.T) {
	db := testdb.Open(t)
	seedPrePorteAccount(t, db)

	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var adopted struct {
		UserID       int64
		PasswordHash string
	}
	err := db.Raw(
		`SELECT user_id, password_hash FROM porte_identities WHERE provider = 'local' AND subject = '1'`,
	).Scan(&adopted).Error
	if err != nil {
		t.Fatalf("read the adopted identity: %v", err)
	}
	if adopted.UserID != 1 {
		t.Fatal("the existing password was not adopted as a local identity")
	}
	if !local.VerifyPassword("a-long-enough-password", adopted.PasswordHash) {
		t.Fatal("the adopted hash does not verify the password it came from")
	}

	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("the adoption is not idempotent: %v", err)
	}
}

// avatar_url arrives on an existing table, so AutoMigrate adds it nullable and
// every row is NULL. Scanning that into a Go string fails, which would take
// out GET /auth/me — every authenticated page — rather than just the avatar.
func TestTheAvatarColumnIsNeverNull(t *testing.T) {
	db := testdb.Open(t)
	if err := db.AutoMigrate(&schemas.User{}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := db.Exec(`ALTER TABLE users ALTER COLUMN avatar_url DROP NOT NULL`).Error; err != nil {
		t.Fatalf("make the column nullable, as AutoMigrate would have: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO users (id, email, name, password_hash, is_admin, created_at, avatar_url)
		 VALUES (1, 'someone@facile.studio', 'Someone', '', true, now(), NULL)`,
	).Error; err != nil {
		t.Fatalf("seed a null avatar: %v", err)
	}

	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var user schemas.User
	if err := db.First(&user, 1).Error; err != nil {
		t.Fatalf("the user row no longer scans: %v", err)
	}
	if user.AvatarURL != "" {
		t.Fatalf("avatar_url = %q, want empty", user.AvatarURL)
	}
}

// porte v0.3.0 keys a password identity on the account id instead of the email
// address, and that re-key is a migration Journal has to carry itself: this app
// runs its own copy of porte's schema against its own users table and never
// calls portepg.EnsureSchema, so a statement living only in porte's pg.Schema
// never reaches this database. Without it porte/local resolves the address to a
// user id, looks the credential up by that id, finds nothing, and every
// existing password login answers "invalid email or password" to the right
// password.
//
// The identity is put back the way v0.2 left it and the migration run again,
// which is both the upgrade a live database performs and the proof that the
// statement is idempotent. The federated subject is the identity provider's and
// must not move with it.
func TestPasswordIdentitiesAreRekeyedOntoTheAccountID(t *testing.T) {
	db := testdb.Open(t)
	seedPrePorteAccount(t, db)
	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	keyItTheOldWay(t, db)

	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("the upgrade migration failed: %v", err)
	}
	assertKeyedOnTheAccountID(t, db)

	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("the re-key is not idempotent: %v", err)
	}
	assertKeyedOnTheAccountID(t, db)
}

// seedPrePorteAccount writes the users row a v0.1 install had: a real argon2id
// hash in users.password_hash, which is what the adoption moves.
func seedPrePorteAccount(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&schemas.User{}); err != nil {
		t.Fatalf("seed the pre-porte users table: %v", err)
	}
	hash, err := local.HashPassword("a-long-enough-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO users (id, email, name, password_hash, is_admin, created_at)
		 VALUES (1, 'someone@facile.studio', 'Someone', ?, true, now())`, hash,
	).Error; err != nil {
		t.Fatalf("seed the account: %v", err)
	}
}

// keyItTheOldWay returns the database to the state porte v0.2 left it in: the
// password identity keyed on the address, beside a federated one keyed on the
// subject its provider issued.
func keyItTheOldWay(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`UPDATE porte_identities SET subject = 'someone@facile.studio' WHERE provider = 'local' AND user_id = 1`,
		`INSERT INTO porte_identities (user_id, provider, subject) VALUES (1, 'https://sso.test/', 'sub-1')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("restore the v0.2 keying: %v", err)
		}
	}
}

// assertKeyedOnTheAccountID is the whole contract: one local identity per
// account, keyed on the id, with the hash it was adopted with, and a federated
// subject the re-key never touched.
func assertKeyedOnTheAccountID(t *testing.T, db *gorm.DB) {
	t.Helper()
	var subject, hash string
	err := db.Raw(`SELECT subject, password_hash FROM porte_identities WHERE provider = 'local' AND user_id = 1`).
		Row().Scan(&subject, &hash)
	if err != nil {
		t.Fatalf("read the local identity: %v", err)
	}
	if subject != "1" {
		t.Fatalf("subject = %q, want the account id: porte v0.3 looks a password up by id, so this credential is unreachable", subject)
	}
	if !local.VerifyPassword("a-long-enough-password", hash) {
		t.Fatal("the re-key lost the password hash it was supposed to carry")
	}

	var locals int64
	if err := db.Raw(`SELECT count(*) FROM porte_identities WHERE provider = 'local'`).Scan(&locals).Error; err != nil {
		t.Fatalf("count the local identities: %v", err)
	}
	if locals != 1 {
		t.Fatalf("%d local identities on one account, so one of them is a password nobody can see", locals)
	}

	var federated string
	if err := db.Raw(`SELECT subject FROM porte_identities WHERE provider = 'https://sso.test/'`).Scan(&federated).Error; err != nil {
		t.Fatalf("read the federated identity: %v", err)
	}
	if federated != "sub-1" {
		t.Fatalf("the re-key moved a federated subject to %q; it belongs to the identity provider", federated)
	}
}
