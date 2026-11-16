package alerts

import (
	"context"
	stderrors "errors"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/netguard"

	"gorm.io/gorm"
)

const maxWindowMinutes = 1440

// Service evaluates and stores alert rules.
type Service struct {
	orm *gorm.DB
}

// NewService wires an alert service onto the database.
func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

// RuleRecord is one alert rule with the name of the saved query it watches.
type RuleRecord struct {
	ID            int64
	WorkspaceID   int64
	Name          string
	SavedQueryID  int64
	QueryName     string
	Provider      schemas.AlertProviderType
	Threshold     int
	WindowMinutes int
	WebhookURL    string
	WebhookHeader *string
	Enabled       bool
	LastFiredAt   *time.Time
	CreatedAt     time.Time
}

func (s *Service) List(ctx context.Context, scope workspace.Scope) ([]RuleRecord, error) {
	var rows []RuleRecord
	err := scope.Where(s.orm.WithContext(ctx).Model(&schemas.AlertRule{}), "alert_rules.workspace_id").
		Select("alert_rules.*, saved_queries.name AS query_name").
		Joins("JOIN saved_queries ON saved_queries.id = alert_rules.saved_query_id").
		Order("alert_rules.created_at desc, alert_rules.id desc").
		Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list alert rules", err)
	}
	return rows, nil
}

func (s *Service) Create(ctx context.Context, owned workspace.Scope, req CreateRequest) (*RuleRecord, error) {
	provider := req.Provider
	if provider == "" {
		provider = string(schemas.AlertProviderWebhook)
	}
	if err := validateRule(req.Name, req.Threshold, req.WindowMinutes, req.WebhookURL, schemas.AlertProviderType(provider)); err != nil {
		return nil, err
	}

	var savedQuery schemas.SavedQuery
	if err := owned.Where(s.orm.WithContext(ctx)).Where("id = ?", req.SavedQueryID).Take(&savedQuery).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("saved query not found")
		}
		return nil, errors.Internal("failed to load saved query", err)
	}

	if schemas.AlertProviderType(provider) == schemas.AlertProviderAntenne && savedQuery.WorkspaceID != schemas.InstanceWorkspaceID {
		return nil, errors.Invalid("the antenne provider is for the instance workspace only; use a webhook")
	}

	rule := schemas.AlertRule{
		WorkspaceID:   savedQuery.WorkspaceID,
		Name:          strings.TrimSpace(req.Name),
		SavedQueryID:  savedQuery.ID,
		Provider:      schemas.AlertProviderType(provider),
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

func (s *Service) SetEnabled(ctx context.Context, scope workspace.Scope, id int64, enabled bool) (*RuleRecord, error) {
	var rule schemas.AlertRule
	if err := scope.Where(s.orm.WithContext(ctx)).Where("id = ?", id).Take(&rule).Error; err != nil {
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

func (s *Service) Delete(ctx context.Context, scope workspace.Scope, id int64) error {
	result := scope.Where(s.orm.WithContext(ctx)).Where("id = ?", id).Delete(&schemas.AlertRule{})
	if result.Error != nil {
		return errors.Internal("failed to delete alert rule", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NotFound("alert rule not found")
	}
	return nil
}

func validateRule(name string, threshold, windowMinutes int, webhookURL string, provider schemas.AlertProviderType) error {
	if strings.TrimSpace(name) == "" {
		return errors.Invalid("name is required")
	}
	if threshold < 1 {
		return errors.Invalid("threshold must be at least 1")
	}
	if windowMinutes < 1 || windowMinutes > maxWindowMinutes {
		return errors.Invalid("window_minutes must be between 1 and 1440")
	}
	if provider == schemas.AlertProviderAntenne {
		return nil
	}
	parsed, err := url.Parse(webhookURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.Invalid("webhook_url must be a valid http or https URL")
	}
	if ip, err := netip.ParseAddr(parsed.Hostname()); err == nil && netguard.IsInternal(ip) {
		return errors.Invalid("webhook_url must not point at a private, loopback, or metadata address")
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

func recordFromRule(rule schemas.AlertRule, queryName string) *RuleRecord {
	return &RuleRecord{
		ID:            rule.ID,
		WorkspaceID:   rule.WorkspaceID,
		Name:          rule.Name,
		SavedQueryID:  rule.SavedQueryID,
		QueryName:     queryName,
		Provider:      rule.Provider,
		Threshold:     rule.Threshold,
		WindowMinutes: rule.WindowMinutes,
		WebhookURL:    rule.WebhookURL,
		WebhookHeader: rule.WebhookHeader,
		Enabled:       rule.Enabled,
		LastFiredAt:   rule.LastFiredAt,
		CreatedAt:     rule.CreatedAt,
	}
}
