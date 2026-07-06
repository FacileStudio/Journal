package alerts

import (
	"context"
	stderrors "errors"
	"net/url"
	"strings"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/gorm"
)

const maxWindowMinutes = 1440

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

type ruleRecord struct {
	ID            int64
	Name          string
	SavedQueryID  int64
	QueryName     string
	Threshold     int
	WindowMinutes int
	WebhookURL    string
	WebhookHeader *string
	Enabled       bool
	LastFiredAt   *time.Time
	CreatedAt     time.Time
}

func (s *Service) List(ctx context.Context) ([]ruleRecord, error) {
	var rows []ruleRecord
	err := s.orm.WithContext(ctx).Model(&schemas.AlertRule{}).
		Select("alert_rules.*, saved_queries.name AS query_name").
		Joins("JOIN saved_queries ON saved_queries.id = alert_rules.saved_query_id").
		Order("alert_rules.created_at desc, alert_rules.id desc").
		Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list alert rules", err)
	}
	return rows, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*ruleRecord, error) {
	if err := validateRule(req.Name, req.Threshold, req.WindowMinutes, req.WebhookURL); err != nil {
		return nil, err
	}

	var savedQuery schemas.SavedQuery
	if err := s.orm.WithContext(ctx).First(&savedQuery, req.SavedQueryID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("saved query not found")
		}
		return nil, errors.Internal("failed to load saved query", err)
	}

	rule := schemas.AlertRule{
		Name:          strings.TrimSpace(req.Name),
		SavedQueryID:  savedQuery.ID,
		Threshold:     req.Threshold,
		WindowMinutes: req.WindowMinutes,
		WebhookURL:    req.WebhookURL,
		WebhookHeader: optionalText(req.WebhookHeader),
		WebhookSecret: optionalText(req.WebhookSecret),
		Enabled:       true,
	}
	if err := s.orm.WithContext(ctx).Create(&rule).Error; err != nil {
		return nil, errors.Internal("failed to store alert rule", err)
	}
	return recordFromRule(rule, savedQuery.Name), nil
}

func (s *Service) SetEnabled(ctx context.Context, id int64, enabled bool) (*ruleRecord, error) {
	var rule schemas.AlertRule
	if err := s.orm.WithContext(ctx).First(&rule, id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("alert rule not found")
		}
		return nil, errors.Internal("failed to load alert rule", err)
	}

	if err := s.orm.WithContext(ctx).Model(&rule).Update("enabled", enabled).Error; err != nil {
		return nil, errors.Internal("failed to update alert rule", err)
	}
	rule.Enabled = enabled

	var savedQuery schemas.SavedQuery
	if err := s.orm.WithContext(ctx).First(&savedQuery, rule.SavedQueryID).Error; err != nil {
		return nil, errors.Internal("failed to load saved query", err)
	}
	return recordFromRule(rule, savedQuery.Name), nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.orm.WithContext(ctx).Delete(&schemas.AlertRule{}, id).Error; err != nil {
		return errors.Internal("failed to delete alert rule", err)
	}
	return nil
}

func validateRule(name string, threshold, windowMinutes int, webhookURL string) error {
	if strings.TrimSpace(name) == "" {
		return errors.Invalid("name is required")
	}
	if threshold < 1 {
		return errors.Invalid("threshold must be at least 1")
	}
	if windowMinutes < 1 || windowMinutes > maxWindowMinutes {
		return errors.Invalid("window_minutes must be between 1 and 1440")
	}
	parsed, err := url.Parse(webhookURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.Invalid("webhook_url must be a valid http or https URL")
	}
	return nil
}

func optionalText(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func recordFromRule(rule schemas.AlertRule, queryName string) *ruleRecord {
	return &ruleRecord{
		ID:            rule.ID,
		Name:          rule.Name,
		SavedQueryID:  rule.SavedQueryID,
		QueryName:     queryName,
		Threshold:     rule.Threshold,
		WindowMinutes: rule.WindowMinutes,
		WebhookURL:    rule.WebhookURL,
		WebhookHeader: rule.WebhookHeader,
		Enabled:       rule.Enabled,
		LastFiredAt:   rule.LastFiredAt,
		CreatedAt:     rule.CreatedAt,
	}
}
