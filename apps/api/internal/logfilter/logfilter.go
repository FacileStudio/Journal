package logfilter

import (
	"time"

	"gorm.io/gorm"
)

// Params is a log query: filters plus the since/until window.
//
// Source filters on meta->>'source', which is what separates entries a server
// shipped from the ones a browser reported. Both land in the same table on
// purpose — one search, one retention, one alert rule — but a stack trace out
// of somebody's browser and a line out of an API are different work, and
// without this the browser half is only reachable by knowing to full-text
// search for it.
type Params struct {
	App    string
	Levels []string
	Query  string

	// Source filters on meta->>'source', which is what separates entries a
	// server shipped from the ones a browser reported. Both land in the same
	// table on purpose — one search, one retention, one alert rule — but a
	// stack trace out of somebody's browser and a line out of an API are
	// different work, and without this the browser half is only reachable by
	// knowing to full-text search for it.
	Source    string
	RequestID string

	Since *time.Time
	Until *time.Time
}

// Apply narrows a log query to the filters in params, in a fixed order:
// app, levels, full-text, source, request id, then the time window.
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
	if params.Source == "server" {
		// "server" is a reserved logical value: the complement of a stamped source.
		// Only the browser ingest stamps meta.source, so this is "everything the
		// server half shipped" — every SDK/collector entry that did not rewrite it.
		query = query.Where("meta->>'source' IS NULL OR meta->>'source' = ''")
	} else if params.Source != "" {
		query = query.Where("meta->>'source' = ?", params.Source)
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
