package mcp

import (
	"net/http"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/schemas"
)

const strangerSub = "stranger-sub"

// stranger is an account that belongs to no workspace yet, which is what an
// MCP client provisioning a brand-new account leaves behind.
func (f *fixture) stranger(t *testing.T) {
	t.Helper()
	var id int64
	must(t, f.db.Raw(`INSERT INTO users (email, name, is_admin) VALUES ('stranger@example.test', 'Stranger', false) RETURNING id`).Scan(&id).Error)
	f.users[strangerSub] = id
}

// A new account's first MCP calls create its personal workspace, and every
// read tool then returns none of the instance workspace's data.
func TestIsolationMCPNewAccountSeesNothing(t *testing.T) {
	f := newFixture(t)
	entry := f.entry(t, "nuage", "error", "instance secret", time.Minute, nil)
	query := f.query(t, "Instance query")
	f.alert(t, "Instance rule", query.ID)
	f.stranger(t)
	session := f.connect(t, strangerSub)
	var workspaces workspacesOutput
	call(t, session, toolWorkspaces, nil, &workspaces)
	if len(workspaces.Workspaces) != 1 || !workspaces.Workspaces[0].Personal || workspaces.Workspaces[0].ID == 1 {
		t.Fatalf("want one personal workspace, got %+v", workspaces)
	}
	var found searchOutput
	call(t, session, toolSearch, map[string]any{"since": "7d"}, &found)
	var counted countOutput
	call(t, session, toolCount, map[string]any{"since": "7d"}, &counted)
	var apps appsOutput
	call(t, session, toolApps, nil, &apps)
	var queries queriesOutput
	call(t, session, toolQueries, nil, &queries)
	var rules alertsOutput
	call(t, session, toolAlerts, nil, &rules)
	if len(found.Entries)+len(apps.Apps)+len(queries.Queries)+len(rules.Alerts) != 0 || counted.Total != 0 {
		t.Fatalf("the new account read instance data: %+v %+v %+v %+v %+v", found, counted, apps, queries, rules)
	}
	refused(t, session, toolRead, map[string]any{"entry_id": entry.ID})
	refused(t, session, toolDeleteQuery, map[string]any{"query_id": query.ID})
	refused(t, session, toolSearch, map[string]any{"workspace_id": 1})
	refused(t, session, toolCreateAlert, map[string]any{"name": "x", "saved_query_id": query.ID, "threshold": 1, "window_minutes": 5, "provider": "webhook", "webhook_url": "https://hooks.example.test/x"})
}

// A deletion requested while the caller was a member is refused at confirm
// time once they no longer belong to the workspace, and the query stays.
func TestIsolationMCPConfirmRechecksTheWorkspace(t *testing.T) {
	f := newFixture(t)
	query := f.query(t, "Shared")
	id := pendingDelete(t, f, memberSub, toolDeleteQuery, map[string]any{"query_id": query.ID})
	must(t, f.db.Exec(`DELETE FROM workspace_members WHERE user_id = ?`, f.users[memberSub]).Error)
	if got := onPage(f, memberSub, "confirm", id); got.Code == http.StatusOK || f.count(&schemas.SavedQuery{}, query.ID) != 1 {
		t.Fatalf("a former member confirmed a deletion: %d %s", got.Code, got.Body)
	}
}
