package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/logfilter"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/gorm"
)

const (
	evaluateInterval  = time.Minute
	webhookTimeout    = 10 * time.Second
	maxPayloadEntries = 5
)

type webhookPayload struct {
	Alert         string             `json:"alert"`
	Query         string             `json:"query"`
	Count         int64              `json:"count"`
	Threshold     int                `json:"threshold"`
	WindowMinutes int                `json:"window_minutes"`
	Since         string             `json:"since"`
	Until         string             `json:"until"`
	Entries       []schemas.LogEntry `json:"entries"`
}

func RunEvaluator(ctx context.Context, orm *gorm.DB, logger *slog.Logger) {
	client := &http.Client{Timeout: webhookTimeout}
	ticker := time.NewTicker(evaluateInterval)
	defer ticker.Stop()
	for {
		evaluateDueRules(ctx, orm, client, logger)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func evaluateDueRules(ctx context.Context, orm *gorm.DB, client *http.Client, logger *slog.Logger) {
	now := time.Now().UTC()

	var rules []schemas.AlertRule
	err := orm.WithContext(ctx).
		Where("enabled AND (last_fired_at IS NULL OR last_fired_at < now() - (window_minutes * interval '1 minute'))").
		Find(&rules).Error
	if err != nil {
		if ctx.Err() == nil {
			logger.Warn("alert rule load failed", slog.Any("error", err))
		}
		return
	}
	if len(rules) == 0 {
		return
	}

	ids := make([]int64, 0, len(rules))
	for _, rule := range rules {
		ids = append(ids, rule.SavedQueryID)
	}
	var savedQueries []schemas.SavedQuery
	if err := orm.WithContext(ctx).Where("id IN ?", ids).Find(&savedQueries).Error; err != nil {
		if ctx.Err() == nil {
			logger.Warn("alert saved query load failed", slog.Any("error", err))
		}
		return
	}
	queriesByID := make(map[int64]schemas.SavedQuery, len(savedQueries))
	for _, savedQuery := range savedQueries {
		queriesByID[savedQuery.ID] = savedQuery
	}

	for _, rule := range rules {
		savedQuery, ok := queriesByID[rule.SavedQueryID]
		if !ok {
			continue
		}
		evaluateRule(ctx, orm, client, logger, rule, savedQuery, now)
	}
}

func evaluateRule(ctx context.Context, orm *gorm.DB, client *http.Client, logger *slog.Logger, rule schemas.AlertRule, savedQuery schemas.SavedQuery, now time.Time) {
	until := now
	since := now.Add(-time.Duration(rule.WindowMinutes) * time.Minute)
	params := logfilter.Params{
		App:       savedQuery.Params.App,
		Levels:    savedQuery.Params.Levels,
		Query:     savedQuery.Params.Q,
		RequestID: savedQuery.Params.RequestID,
		Since:     &since,
		Until:     &until,
	}

	var count int64
	if err := logfilter.Apply(orm.WithContext(ctx).Model(&schemas.LogEntry{}), params).Count(&count).Error; err != nil {
		if ctx.Err() == nil {
			logger.Warn("alert count failed", slog.String("alert", rule.Name), slog.Any("error", err))
		}
		return
	}
	if !shouldFire(rule, now, count) {
		return
	}

	var entries []schemas.LogEntry
	if err := logfilter.Apply(orm.WithContext(ctx).Model(&schemas.LogEntry{}), params).
		Order("created_at desc, id desc").
		Limit(maxPayloadEntries).
		Find(&entries).Error; err != nil {
		if ctx.Err() == nil {
			logger.Warn("alert entry load failed", slog.String("alert", rule.Name), slog.Any("error", err))
		}
		return
	}

	payload := webhookPayload{
		Alert:         rule.Name,
		Query:         savedQuery.Name,
		Count:         count,
		Threshold:     rule.Threshold,
		WindowMinutes: rule.WindowMinutes,
		Since:         since.Format(time.RFC3339),
		Until:         until.Format(time.RFC3339),
		Entries:       entries,
	}
	if !deliverWebhook(ctx, client, logger, rule, payload) {
		return
	}

	if err := orm.WithContext(ctx).Model(&schemas.AlertRule{}).Where("id = ?", rule.ID).Update("last_fired_at", now).Error; err != nil {
		if ctx.Err() == nil {
			logger.Warn("alert last_fired_at update failed", slog.String("alert", rule.Name), slog.Any("error", err))
		}
	}
}

func deliverWebhook(ctx context.Context, client *http.Client, logger *slog.Logger, rule schemas.AlertRule, payload webhookPayload) bool {
	body, err := json.Marshal(payload)
	if err != nil {
		logger.Warn("alert payload encode failed", slog.String("alert", rule.Name), slog.Any("error", err))
		return false
	}

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
	logger.Info("alert fired", slog.String("alert", rule.Name), slog.Int64("count", payload.Count))
	return true
}

func shouldFire(rule schemas.AlertRule, now time.Time, count int64) bool {
	if !rule.Enabled {
		return false
	}
	if count < int64(rule.Threshold) {
		return false
	}
	if rule.LastFiredAt == nil {
		return true
	}
	return !rule.LastFiredAt.After(now.Add(-time.Duration(rule.WindowMinutes) * time.Minute))
}
