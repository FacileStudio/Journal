package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/FacileStudio/Journal/apps/api/modules/antenne"
	"github.com/FacileStudio/Journal/apps/api/schemas"
)

type delivery struct {
	guarded      *http.Client
	trusted      *http.Client
	allowedHosts []string
	antenne      *antenne.Service
}

func (d delivery) clientFor(workspaceID int64, host string) *http.Client {
	if workspaceID == schemas.InstanceWorkspaceID && hostAllowed(host, d.allowedHosts) {
		return d.trusted
	}
	return d.guarded
}

// deliverAntenne marks the rule fired and queues its event in one transaction;
// the outbox drainer sends it. A failed transaction leaves the rule unfired,
// so the next tick tries again.
func deliverAntenne(ctx context.Context, d delivery, logger *slog.Logger, rule schemas.AlertRule, count int64, now time.Time) {
	if err := d.antenne.FireAlert(ctx, rule, count, now); err != nil {
		if ctx.Err() == nil {
			logger.Warn("alert antenne queue failed", slog.String("alert", rule.Name), slog.Any("error", err))
		}
		return
	}
	logger.Info("alert fired via antenne", slog.String("alert", rule.Name), slog.Int64("count", count))
}

func deliverWebhook(ctx context.Context, d delivery, logger *slog.Logger, rule schemas.AlertRule, payload webhookPayload) bool {
	body, err := json.Marshal(payload)
	if err != nil {
		logger.Warn("alert payload encode failed", slog.String("alert", rule.Name), slog.Any("error", err))
		return false
	}

	parsedURL, err := url.Parse(rule.WebhookURL)
	if err != nil {
		logger.Warn("alert webhook url invalid", slog.String("alert", rule.Name), slog.Any("error", err))
		return false
	}
	client := d.clientFor(rule.WorkspaceID, parsedURL.Hostname())

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, rule.WebhookURL, bytes.NewReader(body))
	if err != nil {
		logger.Warn("alert webhook request build failed", slog.String("alert", rule.Name), slog.Any("error", err))
		return false
	}
	request.Header.Set("Content-Type", "application/json")
	if rule.WebhookHeader != nil && rule.WebhookSecret != nil {
		request.Header.Set(*rule.WebhookHeader, *rule.WebhookSecret)
	}

	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() == nil {
			logger.Warn("alert webhook delivery failed", slog.String("alert", rule.Name), slog.Any("error", err))
		}
		return false
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		logger.Warn("alert webhook rejected", slog.String("alert", rule.Name), slog.Int("status", response.StatusCode))
		return false
	}
	logger.Info("alert fired via webhook", slog.String("alert", rule.Name), slog.Int64("count", payload.Count))
	return true
}
