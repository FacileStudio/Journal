package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte/avatarfs"
)

// Erasure is the one auth operation with a law behind it: the account row
// goes, every credential cascades with it — the very token that made the
// request included — and the log entries stay, because they key on app and
// carry nothing of this person. The last-administrator guard is the one
// refusal the endpoint makes.
// TestDeleteMeErasesTheAccountAndItsCredentials runs the delete through the
// real middleware chain and asserts the cascade, the 401 that follows, and
// the guard.
// The first account is an admin and the sole one, so the guard would refuse
// this delete: a successor takes the keys before the door closes.
func TestDeleteMeErasesTheAccountAndItsCredentials(t *testing.T) {
	db := testdb.Migrated(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	appEnv := env.Config{AllowRegistration: true}
	kit, sessions, passwords, avatars := testKitFor(t, appEnv, db, sqlDB)
	router := buildRouter(db, kit, sessions, passwords, avatars, appEnv, slog.New(slog.DiscardHandler), nil)

	registered := call(t, router, http.MethodPost, "/api/auth/register", "", map[string]string{
		"email": "doomed@facile.studio", "name": "Doomed", "password": "a-long-enough-password",
	})
	if registered.Code != http.StatusCreated {
		t.Fatalf("register = %d: %s", registered.Code, registered.Body)
	}
	var created struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(registered.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode register: %v", err)
	}

	successor := &schemas.User{Email: "successor@facile.studio", Name: "Successor", PasswordHash: "x", IsAdmin: true}
	if err := db.Create(successor).Error; err != nil {
		t.Fatalf("seed successor: %v", err)
	}

	if code := call(t, router, http.MethodDelete, "/api/auth/me", created.Token, nil).Code; code != http.StatusNoContent {
		t.Fatalf("delete = %d", code)
	}

	if after := call(t, router, http.MethodGet, "/api/auth/me", created.Token, nil); after.Code != http.StatusUnauthorized {
		t.Fatalf("the token survived the erasure: %d", after.Code)
	}

	var users, sessionRows, identities int64
	db.Model(&schemas.User{}).Where("email <> ?", successor.Email).Count(&users)
	db.Table("porte_sessions").Count(&sessionRows)
	db.Table("porte_identities").Count(&identities)
	if users != 0 || sessionRows != 0 || identities != 0 {
		t.Fatalf("erasure left rows behind: users=%d sessions=%d identities=%d", users, sessionRows, identities)
	}
}

// The last administrator is refused, and promoting a second admin unlocks
// the delete.
func TestDeleteMeSparesTheLastAdministrator(t *testing.T) {
	db := testdb.Migrated(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	appEnv := env.Config{AllowRegistration: true}
	kit, sessions, passwords, avatars := testKitFor(t, appEnv, db, sqlDB)
	router := buildRouter(db, kit, sessions, passwords, avatars, appEnv, slog.New(slog.DiscardHandler), nil)

	register := func(email string) string {
		response := call(t, router, http.MethodPost, "/api/auth/register", "", map[string]string{
			"email": email, "name": "Someone", "password": "a-long-enough-password",
		})
		if response.Code != http.StatusCreated {
			t.Fatalf("register %s = %d: %s", email, response.Code, response.Body)
		}
		var created struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
			t.Fatalf("decode register: %v", err)
		}
		return created.Token
	}

	sole := register("sole@facile.studio")
	register("successor@facile.studio")

	if code := call(t, router, http.MethodDelete, "/api/auth/me", sole, nil).Code; code != http.StatusPreconditionFailed {
		t.Fatalf("the last administrator was not refused: %d", code)
	}

	if err := db.Model(&schemas.User{}).Where("email = ?", "successor@facile.studio").Update("is_admin", true).Error; err != nil {
		t.Fatalf("promote: %v", err)
	}

	if code := call(t, router, http.MethodDelete, "/api/auth/me", sole, nil).Code; code != http.StatusNoContent {
		t.Fatalf("delete with another admin present = %d", code)
	}
}

// The erasure reaches past the database: an SSO avatar cached under
// AVATAR_DIR is personal data too, and it goes when the account does. A
// remote avatar — somebody else's host — must not block the deletion.
func TestDeleteAccountRemovesTheCachedAvatarButNotARemoteOne(t *testing.T) {
	db := testdb.Migrated(t)
	dir := t.TempDir()
	avatars, err := avatarfs.New(dir, "/avatars")
	if err != nil {
		t.Fatalf("avatarfs.New: %v", err)
	}
	avatarURL, err := avatars.Put(context.Background(), "deadbeef", []byte("pretend-png"), "image/png")
	if err != nil {
		t.Fatalf("avatar Put: %v", err)
	}

	local := &schemas.User{Email: "local@facile.studio", Name: "Local", PasswordHash: "x", AvatarURL: avatarURL}
	remote := &schemas.User{Email: "remote@facile.studio", Name: "Remote", PasswordHash: "x", AvatarURL: "https://sso.example/avatar/xyz.png"}
	if err := db.Create(local).Error; err != nil {
		t.Fatalf("seed local: %v", err)
	}
	if err := db.Create(remote).Error; err != nil {
		t.Fatalf("seed remote: %v", err)
	}

	service := auth.NewService(db, nil, removeAvatarFor(avatars))
	ctx := context.Background()
	if err := service.DeleteAccount(ctx, local.ID); err != nil {
		t.Fatalf("delete local: %v", err)
	}
	if err := service.DeleteAccount(ctx, remote.ID); err != nil {
		t.Fatalf("delete remote: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "deadbeef.png")); !os.IsNotExist(err) {
		t.Fatalf("the cached avatar survived the erasure (stat err = %v)", err)
	}
	var count int64
	db.Model(&schemas.User{}).Count(&count)
	if count != 0 {
		t.Fatalf("accounts survived: %d", count)
	}
}
