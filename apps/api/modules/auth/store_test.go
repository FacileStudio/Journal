package auth_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/kit"
)

// TestUpsertFromOIDCStoresThePictureURL pins the avatar to the provider's own
// URL: an https picture is stored as given on every sign-in, and a sign-in that
// carries no https picture clears it, so a removed picture stops showing.
func TestUpsertFromOIDCStoresThePictureURL(t *testing.T) {
	db := testdb.Migrated(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	store := newStore(t, sqlDB)
	ctx := context.Background()
	claims := porte.Claims{
		Provider:      "registre",
		Subject:       "sub-1",
		Email:         "someone@facile.studio",
		EmailVerified: true,
		Picture:       "https://sso.facile.studio/avatars/sub-1",
	}

	steps := []struct {
		picture string
		want    string
	}{
		{"https://sso.facile.studio/avatars/sub-1", "https://sso.facile.studio/avatars/sub-1"},
		{"data:image/svg+xml;base64,AAAA", ""},
	}
	for _, step := range steps {
		claims.Picture = step.picture
		userID, err := store.UpsertFromOIDC(ctx, claims)
		if err != nil {
			t.Fatalf("upsert: %v", err)
		}
		if err := db.Exec(
			`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`,
			userID, claims.Provider, claims.Subject,
		).Error; err != nil {
			t.Fatalf("link the identity: %v", err)
		}
		var user schemas.User
		if err := db.First(&user, userID).Error; err != nil {
			t.Fatalf("read the user: %v", err)
		}
		if user.AvatarURL != step.want {
			t.Fatalf("picture %q stored avatar_url = %q, want %q", step.picture, user.AvatarURL, step.want)
		}
	}
}

// TestBackfillPicturesFillsOnlyEmptyRowsFromThisIssuer seeds three accounts:
// one linked to the issuer with no avatar, one linked with an avatar already,
// and one linked to another provider. Only the first is filled, and a second
// run changes nothing.
func TestBackfillPicturesFillsOnlyEmptyRowsFromThisIssuer(t *testing.T) {
	db := testdb.Migrated(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	store := newStore(t, sqlDB)
	ctx := context.Background()
	issuer := "https://sso.facile.studio"
	if err := db.Exec(
		`INSERT INTO users (id, email, name, password_hash, is_admin, created_at, avatar_url) VALUES
		 (1, 'empty@facile.studio', 'Empty', '', true, now(), ''),
		 (2, 'set@facile.studio', 'Set', '', false, now(), 'https://elsewhere.example/a.png'),
		 (3, 'other@facile.studio', 'Other', '', false, now(), '')`,
	).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO porte_identities (user_id, provider, subject) VALUES
		 (1, ?, 'sub-1'), (2, ?, 'sub-2'), (3, 'https://old.example', 'sub-3')`,
		issuer, issuer,
	).Error; err != nil {
		t.Fatalf("seed identities: %v", err)
	}

	for run, wantRows := range []int64{1, 0} {
		rows, err := store.BackfillPictures(ctx, issuer)
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		if rows != wantRows {
			t.Fatalf("run %d touched %d rows, want %d", run, rows, wantRows)
		}
	}

	want := map[int64]string{
		1: "https://sso.facile.studio/avatars/sub-1",
		2: "https://elsewhere.example/a.png",
		3: "",
	}
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

// porte/kit inserts an account with only the columns it names. Every other
// column of users must come out of the database default, never NULL, or the
// first sign-in leaves a row GORM cannot scan.
func TestAFirstSignInLeavesNoColumnNull(t *testing.T) {
	db := testdb.Migrated(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	userID, err := newStore(t, sqlDB).UpsertFromOIDC(context.Background(), porte.Claims{
		Provider: "https://sso.test", Subject: "new", Email: "new@facile.studio", EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("first sign-in: %v", err)
	}
	var columns []string
	if err := db.Raw(`SELECT column_name FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'users'`).Scan(&columns).Error; err != nil {
		t.Fatalf("list the columns: %v", err)
	}
	for _, column := range columns {
		var null bool
		if err := db.Raw(`SELECT `+column+` IS NULL FROM users WHERE id = ?`, userID).Scan(&null).Error; err != nil {
			t.Fatalf("read %s: %v", column, err)
		}
		if null {
			t.Errorf("users.%s is NULL after a first sign-in through the kit", column)
		}
	}
	var user schemas.User
	if err := db.First(&user, userID).Error; err != nil {
		t.Fatalf("GORM cannot scan the new account: %v", err)
	}
}

func newStore(t *testing.T, sqlDB *sql.DB) *kit.Store {
	t.Helper()
	store, err := auth.NewUserStore(context.Background(), sqlDB)
	if err != nil {
		t.Fatalf("user store: %v", err)
	}
	return store
}

// Step zero of porte's docs/kit.md: every users column the kit does not write
// has a database default, so none is left NULL on a new account. A nullable
// column added without a default fails here, not in production.
func TestNoUsersColumnIsLeftUnwritten(t *testing.T) {
	db := testdb.Migrated(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	if unwritten := newStore(t, sqlDB).Unwritten(); len(unwritten) != 0 {
		t.Fatalf("unwritten = %v, want none", unwritten)
	}
}

// TestEraseAccountDropsPendingMCPActions erases a person with an action
// waiting on the confirm page: their action goes with the account, and
// another person's stays.
func TestEraseAccountDropsPendingMCPActions(t *testing.T) {
	db := testdb.Migrated(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	store := newStore(t, sqlDB)
	if err := db.Exec(`INSERT INTO users (id, email, name) VALUES (1, 'gone@facile.studio', 'Gone'), (2, 'kept@facile.studio', 'Kept')`).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (1, 'https://sso.facile.studio', 'sub-gone')`).Error; err != nil {
		t.Fatalf("link the identity: %v", err)
	}
	if err := db.Exec(`INSERT INTO mcp_actions (id, user_id, kind, target, digest, expires_at) VALUES
		('a1', '1', 'delete_alert', '7', 'd', now() + interval '1 day'),
		('a2', '2', 'delete_alert', '7', 'd', now() + interval '1 day')`).Error; err != nil {
		t.Fatalf("seed actions: %v", err)
	}
	if err := store.EraseAccount(context.Background(), "https://sso.facile.studio", "sub-gone"); err != nil {
		t.Fatalf("erase: %v", err)
	}
	var left []string
	if err := db.Raw(`SELECT id FROM mcp_actions ORDER BY id`).Scan(&left).Error; err != nil {
		t.Fatalf("read actions: %v", err)
	}
	if len(left) != 1 || left[0] != "a2" {
		t.Fatalf("want only the other person's action left, got %v", left)
	}
}
