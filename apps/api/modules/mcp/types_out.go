package mcp

import "github.com/FacileStudio/Journal/apps/api/modules/sourcemaps"

// entryOut is one log entry as a tool returns it.
type entryOut struct {
	ID               int64          `json:"id"`
	App              string         `json:"app"`
	Level            string         `json:"level"`
	Message          string         `json:"message"`
	MessageTruncated bool           `json:"message_truncated,omitempty"`
	CreatedAt        string         `json:"created_at"`
	RequestID        string         `json:"request_id,omitempty"`
	Source           string         `json:"source,omitempty"`
	Meta             map[string]any `json:"meta,omitempty"`
	MetaPreview      string         `json:"meta_preview,omitempty"`
	MetaTruncated    bool           `json:"meta_truncated,omitempty"`
}

// searchOutput is journal_search_logs's result.
type searchOutput struct {
	Since      string     `json:"since"`
	Until      string     `json:"until"`
	Entries    []entryOut `json:"entries"`
	NextCursor string     `json:"next_cursor,omitempty"`
	Note       string     `json:"note,omitempty"`
}

// readOutput is journal_read_log's result.
type readOutput struct {
	Entry      entryOut          `json:"entry"`
	ReceivedAt string            `json:"received_at"`
	SessionID  string            `json:"session_id,omitempty"`
	Newer      []entryOut        `json:"newer,omitempty"`
	Older      []entryOut        `json:"older,omitempty"`
	Stack      *sourcemaps.Stack `json:"stack,omitempty"`
}

// bucketOut is the per-level count of one time bucket.
type bucketOut struct {
	Start  string           `json:"start"`
	Counts map[string]int64 `json:"counts"`
}

// countOutput is journal_count_logs's result.
type countOutput struct {
	Since         string           `json:"since"`
	Until         string           `json:"until"`
	BucketSeconds int64            `json:"bucket_seconds"`
	Total         int64            `json:"total"`
	ByLevel       map[string]int64 `json:"by_level"`
	Buckets       []bucketOut      `json:"buckets"`
}

// appOut is one app that has logged.
type appOut struct {
	Name     string `json:"name"`
	Entries  int64  `json:"entries"`
	LastSeen string `json:"last_seen"`
}

// appsOutput is journal_list_apps's result.
type appsOutput struct {
	Apps []appOut `json:"apps"`
}

// queryOut is one saved query.
type queryOut struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	App       string   `json:"app,omitempty"`
	Levels    []string `json:"levels,omitempty"`
	Q         string   `json:"q,omitempty"`
	Source    string   `json:"source,omitempty"`
	RequestID string   `json:"request_id,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// queriesOutput is journal_list_saved_queries's result.
type queriesOutput struct {
	Queries []queryOut `json:"queries"`
}

// queryOutput is journal_save_query's result.
type queryOutput struct {
	Status string   `json:"status"`
	Query  queryOut `json:"query"`
}

// alertOut is one alert rule. A webhook shows its host only: the full URL of
// a chat or incident webhook is a credential.
type alertOut struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	SavedQueryID  int64  `json:"saved_query_id"`
	SavedQuery    string `json:"saved_query"`
	Threshold     int    `json:"threshold"`
	WindowMinutes int    `json:"window_minutes"`
	Provider      string `json:"provider"`
	WebhookHost   string `json:"webhook_host,omitempty"`
	Enabled       bool   `json:"enabled"`
	LastFiredAt   string `json:"last_fired_at,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// alertsOutput is journal_list_alerts's result.
type alertsOutput struct {
	Alerts []alertOut `json:"alerts"`
}

// alertOutput is the result of a tool that changed one alert rule.
type alertOutput struct {
	Status string   `json:"status"`
	Alert  alertOut `json:"alert"`
}

// deleteOutput is the result of a confirmed deletion tool.
type deleteOutput struct {
	Status     string `json:"status"`
	ID         int64  `json:"id,omitempty"`
	ConfirmURL string `json:"confirm_url,omitempty"`
	Message    string `json:"message,omitempty"`
}
