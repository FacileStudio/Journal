package logfilter

import (
	"time"

	"gorm.io/gorm"
)

type Params struct {
	App       string
	Levels    []string
	Query     string
	RequestID string
	Since     *time.Time
	Until     *time.Time
}

func Apply(query *gorm.DB, params Params) *gorm.DB {
	if params.App != "" {
		query = query.Where("app = ?", params.App)
	}
	if len(params.Levels) > 0 {
		query = query.Where("level IN ?", params.Levels)
	}
	if params.Query != "" {
		query = query.Where("search @@ websearch_to_tsquery('simple', ?)", params.Query)
	}
	if params.RequestID != "" {
		query = query.Where("meta->>'request_id' = ?", params.RequestID)
	}
	if params.Since != nil {
		query = query.Where("created_at >= ?", *params.Since)
	}
	if params.Until != nil {
		query = query.Where("created_at <= ?", *params.Until)
	}
	return query
}
