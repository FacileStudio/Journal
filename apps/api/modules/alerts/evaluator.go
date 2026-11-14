package alerts

import (
	"context"
	"log/slog"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/logfilter"
	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/Journal/apps/api/modules/antenne"
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

// RunEvaluator scans enabled alert rules on a loop and delivers when one fires,
// until ctx is cancelled.
func RunEvaluator(ctx context.Context, orm *gorm.DB, logger *slog.Logger, allowedHosts []string, antenneService *antenne.Service) {
	d := delivery{
		guarded:      guardedClient(webhookTimeout),
		trusted:      trustedClient(webhookTimeout),
		allowedHosts: allowedHosts,
		antenne:      antenneService,
	}
	ticker := time.NewTicker(evaluateInterval)
	defer ticker.Stop()
	for {
		evaluateDueRules(ctx, orm, d, logger)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func evaluateDueRules(ctx context.Context, orm *gorm.DB, d delivery, logger *slog.Logger) {
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
		if !ok || savedQuery.WorkspaceID != rule.WorkspaceID {
			continue
		}
		evaluateRule(ctx, orm, d, logger, rule, savedQuery, now)
	}
}

func evaluateRule(ctx context.Context, orm *gorm.DB, d delivery, logger *slog.Logger, rule schemas.AlertRule, savedQuery schemas.SavedQuery, now time.Time) {
	until := now
	since := now.Add(-time.Duration(rule.WindowMinutes) * time.Minute)
	params := logfilter.Params{
		App:       savedQuery.Params.App,
		Levels:    savedQuery.Params.Levels,
		Query:     savedQuery.Params.Q,
		Source:    savedQuery.Params.Source,
		RequestID: savedQuery.Params.RequestID,
		Since:     &since,
		Until:     &until,
	}

	own := workspace.Of(rule.WorkspaceID)
	var count int64
	if err := logfilter.Apply(orm.WithContext(ctx), own, params).Count(&count).Error; err != nil {
		if ctx.Err() == nil {
			logger.Warn("alert count failed", slog.String("alert", rule.Name), slog.Any("error", err))
		}
		return
	}
	if !shouldFire(rule, now, count) {
		return
	}

	var entries []schemas.LogEntry
	if err := logfilter.Apply(orm.WithContext(ctx), own, params).
		Order("created_at desc, id desc").
		Limit(maxPayloadEntries).
		Find(&entries).Error; err != nil {
		if ctx.Err() == nil {
			logger.Warn("alert entry load failed", slog.String("alert", rule.Name), slog.Any("error", err))
		}
		return
	}

	if rule.Provider == schemas.AlertProviderAntenne && rule.WorkspaceID != schemas.InstanceWorkspaceID {
		logger.Warn("antenne alert outside the instance workspace skipped", slog.String("alert", rule.Name))
		return
	}
	if rule.Provider == schemas.AlertProviderAntenne && d.antenne != nil {
		deliverAntenne(ctx, d, logger, rule, count, now)
		return
	}
	if rule.Provider == schemas.AlertProviderAntenne {
		logger.Warn("antenne service not available", slog.String("alert", rule.Name))
	} else {
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
		if !deliverWebhook(ctx, d, logger, rule, payload) {
			return
		}
	}

	if err := orm.WithContext(ctx).Model(&schemas.AlertRule{}).Where("id = ?", rule.ID).Update("last_fired_at", now).Error; err != nil {
		if ctx.Err() == nil {
			logger.Warn("alert last_fired_at update failed", slog.String("alert", rule.Name), slog.Any("error", err))
		}
	}
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
