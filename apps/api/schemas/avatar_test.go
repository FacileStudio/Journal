package schemas_test

import (
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/schemas"
)

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

// A row from before the switch points at Journal's own copy under /avatars,
// which is no longer served. Migrate clears it, so the dashboard draws the
// initial instead of a broken image, and keeps a provider's https URL.
func TestMigrateClearsLocalAvatarCopies(t *testing.T) {
	db := testdb.Migrated(t)
	if err := db.Exec(
		`INSERT INTO users (id, email, name, password_hash, is_admin, created_at, avatar_url) VALUES
		 (1, 'copy@facile.studio', 'Copy', '', true, now(), '/avatars/deadbeef.png'),
		 (2, 'stable@facile.studio', 'Stable', '', false, now(), 'https://sso.facile.studio/avatars/abc')`,
	).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	want := map[int64]string{1: "", 2: "https://sso.facile.studio/avatars/abc"}
	for id, url := range want {
		var user schemas.User
		if err := db.First(&user, id).Error; err != nil {
			t.Fatalf("read user %d: %v", id, err)
		}
		if user.AvatarURL != url {
			t.Fatalf("user %d avatar_url = %q, want %q", id, user.AvatarURL, url)
		}
	}
}
