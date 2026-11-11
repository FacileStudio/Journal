package mcp

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/kit"
	"github.com/go-chi/chi/v5"
)

// pendingDelete asks a client that cannot confirm to run tool on args, and
// returns the action id.
func pendingDelete(t *testing.T, f *fixture, subject, tool string, args map[string]any) string {
	t.Helper()
	var out deleteOutput
	call(t, f.connect(t, subject), tool, args, &out)
	if out.Status != "pending_confirmation" || out.ConfirmURL == "" {
		t.Fatalf("want a confirm link, got %+v", out)
	}
	return actionID(out.ConfirmURL)
}

// onPage calls a confirm page route as subject's own session, or as a
// delegated token when actor is set. An empty route lists the actions.
func onPage(f *fixture, subject, route, id, actor string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if route != "" {
		req = httptest.NewRequest(http.MethodPost, "/"+id+"/"+route, nil)
	}
	userID := f.users[subject]
	ctx := kit.WithIdentity(req.Context(), kit.Identity{UserID: userID})
	ctx = porte.WithIdentity(ctx, porte.Identity{UserID: userID, Actor: actor})
	router := chi.NewRouter()
	f.service.page.Routes(router)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req.WithContext(ctx))
	return recorder
}

// TestConfirmPageDeletesQueryOnce checks the person confirms in Journal, the
// query goes once, a second confirmation finds nothing, an agent cannot
// confirm its own request and another person does not see it.
func TestConfirmPageDeletesQueryOnce(t *testing.T) {
	f := newFixture(t)
	q := f.query(t, "To delete")
	id := pendingDelete(t, f, memberSub, toolDeleteQuery, map[string]any{"query_id": q.ID})
	if got := onPage(f, memberSub, "confirm", id, "bulle"); got.Code != http.StatusForbidden || f.count(&schemas.SavedQuery{}, q.ID) != 1 {
		t.Fatalf("a delegated token must not confirm, got %d", got.Code)
	}
	if other := onPage(f, adminSub, "", "", ""); strings.Contains(other.Body.String(), id) {
		t.Fatalf("another person must not see the action, got %s", other.Body)
	}
	list := onPage(f, memberSub, "", "", "")
	if !strings.Contains(list.Body.String(), `"state":"ready"`) || !strings.Contains(list.Body.String(), `"query":{"id":`+strconv.FormatInt(q.ID, 10)) {
		t.Fatalf("want one ready action with its query, got %d %s", list.Code, list.Body)
	}
	if got := onPage(f, memberSub, "confirm", id, ""); got.Code != http.StatusOK || f.count(&schemas.SavedQuery{}, q.ID) != 0 {
		t.Fatalf("want the query deleted, got %d %s", got.Code, got.Body)
	}
	if again := onPage(f, memberSub, "confirm", id, ""); again.Code != http.StatusNotFound {
		t.Fatalf("a confirmation is single use, got %d", again.Code)
	}
}

// TestEditedOrDemotedNeverDeletes checks an alert rule edited after the
// request is kept, and that a person who lost administrator rights can no
// longer confirm a rule's deletion.
func TestEditedOrDemotedNeverDeletes(t *testing.T) {
	f := newFixture(t)
	q := f.query(t, "Watched")
	edited := f.alert(t, "Edited", q.ID)
	id := pendingDelete(t, f, adminSub, toolDeleteAlert, map[string]any{"alert_id": edited.ID})
	must(t, f.db.Model(&schemas.AlertRule{}).Where("id = ?", edited.ID).Update("threshold", 99).Error)
	if got := onPage(f, adminSub, "confirm", id, ""); got.Code != http.StatusPreconditionFailed || f.count(&schemas.AlertRule{}, edited.ID) != 1 {
		t.Fatalf("want 412 and the rule kept, got %d %s", got.Code, got.Body)
	}
	kept := f.alert(t, "Kept", q.ID)
	id = pendingDelete(t, f, adminSub, toolDeleteAlert, map[string]any{"alert_id": kept.ID})
	must(t, f.db.Exec(`UPDATE users SET is_admin = false WHERE id = ?`, f.users[adminSub]).Error)
	if got := onPage(f, adminSub, "confirm", id, ""); got.Code == http.StatusOK || f.count(&schemas.AlertRule{}, kept.ID) != 1 {
		t.Fatalf("a demoted person must not delete the rule, got %d %s", got.Code, got.Body)
	}
	if list := onPage(f, adminSub, "", "", ""); strings.Contains(list.Body.String(), `"alert":`) {
		t.Fatalf("a demoted person must not read the rule on the page, got %s", list.Body)
	}
}
