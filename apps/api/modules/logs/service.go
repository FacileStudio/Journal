package logs

import (
	"context"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

type ListParams struct {
	App    string
	Levels []string
	Query  string
	Since  *time.Time
	Until  *time.Time
	Limit  int
	Before *int64
}

func (s *Service) List(ctx context.Context, params ListParams) ([]schemas.LogEntry, error) {
	if params.Limit <= 0 {
		params.Limit = 100
	}
	if params.Limit > 1000 {
		params.Limit = 1000
	}

	query := s.orm.WithContext(ctx).Model(&schemas.LogEntry{})
	if params.App != "" {
		query = query.Where("app = ?", params.App)
	}
	if len(params.Levels) > 0 {
		query = query.Where("level IN ?", params.Levels)
	}
	if params.Query != "" {
		query = query.Where("search @@ websearch_to_tsquery('simple', ?)", params.Query)
	}
	if params.Since != nil {
		query = query.Where("created_at >= ?", *params.Since)
	}
	if params.Until != nil {
		query = query.Where("created_at <= ?", *params.Until)
	}
	if params.Before != nil {
		query = query.Where("id < ?", *params.Before)
	}

	var records []schemas.LogEntry
	if err := query.Order("created_at desc, id desc").Limit(params.Limit).Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list log entries", err)
	}
	return records, nil
}

type appRow struct {
	Name     string
	Count    int64
	LastSeen time.Time
}

func (s *Service) Apps(ctx context.Context) ([]AppSummary, error) {
	var rows []appRow
	if err := s.orm.WithContext(ctx).Model(&schemas.LogEntry{}).
		Select("app as name, count(*) as count, max(created_at) as last_seen").
		Group("app").
		Order("last_seen desc").
		Scan(&rows).Error; err != nil {
		return nil, errors.Internal("failed to list apps", err)
	}

	apps := make([]AppSummary, 0, len(rows))
	for _, row := range rows {
		apps = append(apps, AppSummary{
			Name:     row.Name,
			Count:    row.Count,
			LastSeen: row.LastSeen.UTC().Format(time.RFC3339),
		})
	}
	return apps, nil
}
