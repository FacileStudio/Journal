package mcp

import (
	"context"
	"net/http"
	"strconv"

	"github.com/FacileStudio/porte/mcpauth"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// webActor marks, in the audit, a confirmation made on the confirm page.
const webActor = "journal-web"

// pendingUser is the signed-in person of a confirm page request. Only the
// person's own browser session passes: a bearer, a CLI or mobile session, an
// API key or a delegated token is refused, so an agent never confirms its own
// request.
func pendingUser(r *http.Request) (string, error) {
	identity, err := mcpauth.RequireOwnSession(r)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(identity.UserID, 10), nil
}

// describe adds the saved query or the alert rule an action would delete, as
// it is now, to a confirm page item. A rule is described only to someone who
// may still read it.
func (t *toolset) describe(ctx context.Context, item mcpkit.PendingItem, a mcpkit.Action) PendingItem {
	out := PendingItem{PendingItem: item}
	id, err := idOf(a)
	if err != nil {
		return out
	}
	switch a.Kind {
	case kindDeleteQuery:
		if q, err := t.savedQuery(ctx, id); err == nil {
			out.Query = &PendingQuery{ID: q.ID, Name: q.Name, App: q.Params.App, Levels: q.Params.Levels, Q: q.Params.Q}
		}
	case kindDeleteAlert:
		if !t.isAdmin(ctx, a.UserID) {
			return out
		}
		if r, err := t.alertRule(ctx, id); err == nil {
			out.Alert = &PendingAlert{
				ID: r.ID, Name: r.Name, SavedQuery: r.QueryName, Threshold: r.Threshold, WindowMinutes: r.WindowMinutes,
				Provider: string(r.Provider), Enabled: r.Enabled,
			}
		}
	}
	return out
}

// done is the confirm page's answer once a deletion ran.
func done(context.Context, mcpkit.Outcome) mcpkit.PendingResult {
	return mcpkit.PendingResult{Status: statusDeleted}
}
