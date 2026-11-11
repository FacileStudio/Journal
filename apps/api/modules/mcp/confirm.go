package mcp

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/FacileStudio/Journal/apps/api/modules/alerts"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// The sensitive actions: deleting a saved query or an alert rule, neither of
// which can be undone.
const (
	kindDeleteQuery = "delete_saved_query"
	kindDeleteAlert = "delete_alert"
)

var (
	errTarget     = errors.NotFound("this action names no saved query or alert rule")
	errAdminGone  = errors.Forbidden("only a Journal administrator can delete an alert rule")
	errStillWatch = errors.Failed("an alert rule now watches this saved query, so it cannot be deleted")
)

// kinds are the sensitive actions. Each fingerprint covers the target as it
// was when the person was asked, so an edit in between closes the action
// instead of deleting something the person did not see.
func (t *toolset) kinds() map[string]mcpkit.Kind {
	return map[string]mcpkit.Kind{
		kindDeleteQuery: {Tool: toolDeleteQuery, Fingerprint: t.queryPrint, Run: t.runDeleteQuery},
		kindDeleteAlert: {Tool: toolDeleteAlert, Fingerprint: t.alertPrint, Run: t.runDeleteAlert},
	}
}

// idOf parses an action's target id.
func idOf(a mcpkit.Action) (int64, error) {
	_, id, err := mcpkit.TargetIDs(a, errTarget)
	return id, err
}

// queryPrint digests the saved query an action targets, as it is now.
func (t *toolset) queryPrint(ctx context.Context, a mcpkit.Action) (string, error) {
	id, err := idOf(a)
	if err != nil {
		return "", err
	}
	query, err := t.savedQuery(ctx, id)
	if err != nil {
		return "", err
	}
	params, _ := json.Marshal(query.Params)
	return mcpkit.Digest(strconv.FormatInt(query.ID, 10), query.Name, string(params), query.CreatedAt.UTC().Format(time.RFC3339Nano)), nil
}

// alertPrint digests the alert rule an action targets, as it is now. A
// person who lost administrator rights since asking cannot confirm.
func (t *toolset) alertPrint(ctx context.Context, a mcpkit.Action) (string, error) {
	if !t.isAdmin(ctx, a.UserID) {
		return "", errAdminGone
	}
	id, err := idOf(a)
	if err != nil {
		return "", err
	}
	rule, err := t.alertRule(ctx, id)
	if err != nil {
		return "", err
	}
	return ruleDigest(rule), nil
}

// ruleDigest fingerprints an alert rule.
func ruleDigest(r alerts.RuleRecord) string {
	return mcpkit.Digest(
		strconv.FormatInt(r.ID, 10), r.Name, strconv.FormatInt(r.SavedQueryID, 10), string(r.Provider),
		strconv.Itoa(r.Threshold), strconv.Itoa(r.WindowMinutes), r.WebhookURL, strconv.FormatBool(r.Enabled),
	)
}

// runDeleteQuery deletes the targeted saved query, refused when an alert
// rule started watching it since the question.
func (t *toolset) runDeleteQuery(ctx context.Context, a mcpkit.Action) error {
	id, err := idOf(a)
	if err != nil {
		return err
	}
	if n, err := t.watchers(ctx, id); err != nil || n > 0 {
		return errStillWatch
	}
	return t.Queries.Delete(ctx, id)
}

// runDeleteAlert deletes the targeted alert rule.
func (t *toolset) runDeleteAlert(ctx context.Context, a mcpkit.Action) error {
	if !t.isAdmin(ctx, a.UserID) {
		return errAdminGone
	}
	id, err := idOf(a)
	if err != nil {
		return err
	}
	return t.Alerts.Delete(ctx, id)
}
