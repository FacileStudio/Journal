package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/schemas"
)

// Erasure is the one auth operation with a law behind it: the account row
// goes, every credential cascades with it — the very token that made the
// request included — and the log entries stay, because they key on app and
// carry nothing of this person. The last-administrator guard is the one
// refusal the endpoint makes.
// TestDeleteMeErasesTheAccountAndItsCredentials runs the delete through the
// real middleware chain and asserts the cascade, the 401 that follows, and
// the guard.
func TestDeleteMeErasesTheAccountAndItsCredentials(t *testing.T) {
	db := testdb.Migrated(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	appEnv := env.Config{AllowRegistration: true}
	kit, sessions, passwords, avatars := testKitFor(t, appEnv, db, sqlDB)
	router := buildRouter(db, kit, sessions, passwords, avatars, appEnv, slog.New(slog.DiscardHandler))

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

	// The first account is an admin, and the sole one, so the guard would
	// refuse this delete. A successor takes the keys before the door closes.
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

	var users, sessions2, identities int64
	db.Model(&schemas.User{}).Where("email <> ?", successor.Email).Count(&users)
	db.Table("porte_sessions").Count(&sessions2)
	db.Table("porte_identities").Count(&identities)
	if users != 0 || sessions2 != 0 || identities != 0 {
		t.Fatalf("erasure left rows behind: users=%d sessions=%d identities=%d", users, sessions2, identities)
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
	router := buildRouter(db, kit, sessions, passwords, avatars, appEnv, slog.New(slog.DiscardHandler))

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
