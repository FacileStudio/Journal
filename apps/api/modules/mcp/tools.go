package mcp

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/mcpauth"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	toolSearch      = "journal_search_logs"
	toolRead        = "journal_read_log"
	toolCount       = "journal_count_logs"
	toolApps        = "journal_list_apps"
	toolQueries     = "journal_list_saved_queries"
	toolSaveQuery   = "journal_save_query"
	toolDeleteQuery = "journal_delete_saved_query"
	toolAlerts      = "journal_list_alerts"
	toolCreateAlert = "journal_create_alert"
	toolSetAlert    = "journal_set_alert_enabled"
	toolDeleteAlert = "journal_delete_alert"
	toolWorkspaces  = "journal_list_workspaces"
	hintEntry       = "Call journal_search_logs for the ids of entries that exist."
	hintQuery       = "Call journal_list_saved_queries for the ids of saved queries."
	hintAlert       = "Call journal_list_alerts for the ids of alert rules."
	statusDeleted   = "deleted"
)

var (
	errNoCaller = stderrors.New("this request carries no valid access token. Reconnect Journal")
)

// toolset holds what the tool handlers share.
type toolset struct {
	Deps
	confirm *mcpkit.Confirmations
}

// callerOf reads the Journal account behind the verified token, refused the
// way kit.RequireAuth refuses it: a deleted account never acts.
func (t *toolset) callerOf(ctx context.Context, req *sdk.CallToolRequest) (kit.Identity, error) {
	account, err := mcpauth.AccountExists(ctx, req.Extra.TokenInfo, t.Accounts.Identity)
	if err != nil {
		return kit.Identity{}, errNoCaller
	}
	return account, nil
}

// adminOf is callerOf for the alert tools, which middleware.RequireAdmin
// guards on the REST side.
func target(id int64) string {
	return strconv.FormatInt(id, 10)
}
