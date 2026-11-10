package auth_test

import (
	"context"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte"
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
	store := auth.NewUserStore(sqlDB)
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
	store := auth.NewUserStore(sqlDB)
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
