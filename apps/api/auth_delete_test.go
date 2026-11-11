package main

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/schemas"
)

// Self-service erasure: the account row goes, every credential cascades with
// it, the very token that made the request included. The last-administrator
// guard is the one refusal the endpoint makes.
func TestDeleteMeErasesTheAccountAndItsCredentials(t *testing.T) {
	app := newLiveApp(t)
	doomed, token := app.signIn(t, "doomed@facile.studio", false)
	app.signIn(t, "admin@facile.studio", true)
	if err := app.db.Exec(`INSERT INTO mcp_actions (id, user_id, kind, target, digest, expires_at)
		VALUES ('doomed-action', ?, 'delete_alert', '1', 'd', now() + interval '1 day')`, strconv.FormatInt(doomed, 10)).Error; err != nil {
		t.Fatalf("seed a pending action: %v", err)
	}

	if code := call(t, app.router, http.MethodDelete, "/api/auth/me", token, nil).Code; code != http.StatusNoContent {
		t.Fatalf("delete = %d", code)
	}
	if after := call(t, app.router, http.MethodGet, "/api/auth/me", token, nil); after.Code != http.StatusUnauthorized {
		t.Fatalf("the token survived the erasure: %d", after.Code)
	}

	var users, sessionRows, actions int64
	app.db.Model(&schemas.User{}).Where("email = ?", "doomed@facile.studio").Count(&users)
	app.db.Table("porte_sessions").Where("user_id = ?", doomed).Count(&sessionRows)
	app.db.Table("mcp_actions").Where("id = ?", "doomed-action").Count(&actions)
	if users != 0 || sessionRows != 0 || actions != 0 {
		t.Fatalf("erasure left rows behind: users=%d sessions=%d mcp_actions=%d", users, sessionRows, actions)
	}
}

// The last administrator is refused, and promoting a second admin unlocks the
// delete.
func TestDeleteMeSparesTheLastAdministrator(t *testing.T) {
	app := newLiveApp(t)
	_, sole := app.signIn(t, "sole@facile.studio", true)
	app.signIn(t, "successor@facile.studio", false)

	if code := call(t, app.router, http.MethodDelete, "/api/auth/me", sole, nil).Code; code != http.StatusPreconditionFailed {
		t.Fatalf("the last administrator was not refused: %d", code)
	}
	err := app.db.Model(&schemas.User{}).Where("email = ?", "successor@facile.studio").Update("is_admin", true).Error
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if code := call(t, app.router, http.MethodDelete, "/api/auth/me", sole, nil).Code; code != http.StatusNoContent {
		t.Fatalf("delete with another admin present = %d", code)
	}
}
