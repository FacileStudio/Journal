package mcp

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/url"

	"github.com/FacileStudio/Journal/apps/api/modules/alerts"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	errAlertMissing = errors.NotFound("alert rule not found")
	errProvider     = stderrors.New(`provider must be "antenne" or "webhook"`)
)

// listAlerts is journal_list_alerts.
func (t *toolset) listAlerts(ctx context.Context, req *sdk.CallToolRequest, _ alertsInput) (*sdk.CallToolResult, alertsOutput, error) {
	if _, err := t.adminOf(ctx, req); err != nil {
		return nil, alertsOutput{}, err
	}
	rules, err := t.Alerts.List(ctx)
	if err != nil {
		return nil, alertsOutput{}, mcpkit.Failure(err, "")
	}
	out := alertsOutput{Alerts: make([]alertOut, len(rules))}
	for i, rule := range rules {
		out.Alerts[i] = alertView(rule)
	}
	return nil, out, nil
}

// createAlert is journal_create_alert.
func (t *toolset) createAlert(ctx context.Context, req *sdk.CallToolRequest, in createAlertInput) (*sdk.CallToolResult, alertOutput, error) {
	if _, err := t.adminOf(ctx, req); err != nil {
		return nil, alertOutput{}, err
	}
	provider := in.Provider
	if provider == "" {
		provider = string(schemas.AlertProviderAntenne)
	}
	if provider != string(schemas.AlertProviderAntenne) && provider != string(schemas.AlertProviderWebhook) {
		return nil, alertOutput{}, errProvider
	}
	rule, err := t.Alerts.Create(ctx, alerts.CreateRequest{
		Name: in.Name, SavedQueryID: in.SavedQueryID, Provider: provider, Threshold: in.Threshold,
		WindowMinutes: in.WindowMinutes, WebhookURL: in.WebhookURL,
	})
	if err != nil {
		return nil, alertOutput{}, mcpkit.Failure(err, hintQuery)
	}
	return nil, alertOutput{Status: "created", Alert: alertView(*rule)}, nil
}

// setAlert is journal_set_alert_enabled.
func (t *toolset) setAlert(ctx context.Context, req *sdk.CallToolRequest, in setAlertInput) (*sdk.CallToolResult, alertOutput, error) {
	if _, err := t.adminOf(ctx, req); err != nil {
		return nil, alertOutput{}, err
	}
	rule, err := t.Alerts.SetEnabled(ctx, in.AlertID, in.Enabled)
	if err != nil {
		return nil, alertOutput{}, mcpkit.Failure(err, hintAlert)
	}
	status := "disabled"
	if rule.Enabled {
		status = "enabled"
	}
	return nil, alertOutput{Status: status, Alert: alertView(*rule)}, nil
}

// deleteAlert is journal_delete_alert: the rule goes once the user confirms.
func (t *toolset) deleteAlert(ctx context.Context, req *sdk.CallToolRequest, in alertRef) (*sdk.CallToolResult, deleteOutput, error) {
	if mcpkit.Retrying(req) {
		return t.resume(ctx, req, kindDeleteAlert, "alert rule")
	}
	if _, err := t.adminOf(ctx, req); err != nil {
		return nil, deleteOutput{}, err
	}
	rule, err := t.alertRule(ctx, in.AlertID)
	if err != nil {
		return nil, deleteOutput{}, mcpkit.Failure(err, hintAlert)
	}
	question := fmt.Sprintf("Delete the alert rule %q, which watches the saved query %q? This cannot be undone.", rule.Name, rule.QueryName)
	return t.request(ctx, req, kindDeleteAlert, rule.ID, question)
}

// alertRule finds one alert rule by id.
func (t *toolset) alertRule(ctx context.Context, id int64) (alerts.RuleRecord, error) {
	rules, err := t.Alerts.List(ctx)
	if err != nil {
		return alerts.RuleRecord{}, err
	}
	for _, rule := range rules {
		if rule.ID == id {
			return rule, nil
		}
	}
	return alerts.RuleRecord{}, errAlertMissing
}

// alertView renders an alert rule, naming a webhook by its host only.
func alertView(r alerts.RuleRecord) alertOut {
	out := alertOut{
		ID: r.ID, Name: r.Name, SavedQueryID: r.SavedQueryID, SavedQuery: r.QueryName, Threshold: r.Threshold,
		WindowMinutes: r.WindowMinutes, Provider: string(r.Provider), Enabled: r.Enabled, CreatedAt: stamp(r.CreatedAt),
	}
	if r.Provider == schemas.AlertProviderWebhook {
		if parsed, err := url.Parse(r.WebhookURL); err == nil {
			out.WebhookHost = parsed.Host
		}
	}
	if r.LastFiredAt != nil {
		out.LastFiredAt = stamp(*r.LastFiredAt)
	}
	return out
}
