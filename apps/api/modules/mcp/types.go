package mcp

// filterInput is the log filter the search and count tools share.
type filterInput struct {
	App       string   `json:"app,omitempty" jsonschema:"only this app, a name from journal_list_apps"`
	Levels    []string `json:"levels,omitempty" jsonschema:"only these levels: debug, info, warn, error"`
	Q         string   `json:"q,omitempty" jsonschema:"full-text search on the message: words, \"a phrase\", -excluded"`
	Source    string   `json:"source,omitempty" jsonschema:"server for what apps shipped, browser for what visitors' browsers reported"`
	RequestID string   `json:"request_id,omitempty" jsonschema:"only entries with this meta.request_id, one call across services"`
	SessionID string   `json:"session_id,omitempty" jsonschema:"only entries with this meta.session_id, one browser tab"`
	Since     string   `json:"since,omitempty" jsonschema:"start of the window: an RFC 3339 time or a duration back from now such as 15m, 6h, 7d. Default 24h"`
	Until     string   `json:"until,omitempty" jsonschema:"end of the window, same forms. Default now"`
}

// searchInput is journal_search_logs's input.
type searchInput struct {
	filterInput
	Limit          int    `json:"limit,omitempty" jsonschema:"entries per page, 1 to 200. Default 50"`
	Cursor         string `json:"cursor,omitempty" jsonschema:"next_cursor from the previous page"`
	ResponseFormat string `json:"response_format,omitempty" jsonschema:"concise (default) or detailed, which adds each entry's meta"`
}

// readInput is journal_read_log's input.
type readInput struct {
	EntryID      int64 `json:"entry_id" jsonschema:"an entry id from journal_search_logs"`
	Before       int   `json:"before,omitempty" jsonschema:"how many older entries to add around it, 0 to 50. Default 0"`
	After        int   `json:"after,omitempty" jsonschema:"how many newer entries to add around it, 0 to 50. Default 0"`
	ResolveStack bool  `json:"resolve_stack,omitempty" jsonschema:"map meta.stack through the source maps uploaded for meta.release"`
}

// countInput is journal_count_logs's input.
type countInput struct {
	filterInput
}

// appsInput is journal_list_apps's input, which takes nothing.
type appsInput struct{}

// queriesInput is journal_list_saved_queries's input, which takes nothing.
type queriesInput struct{}

// saveQueryInput is journal_save_query's input.
type saveQueryInput struct {
	Name      string   `json:"name" jsonschema:"a unique name, such as Checkout errors"`
	App       string   `json:"app,omitempty" jsonschema:"only this app"`
	Levels    []string `json:"levels,omitempty" jsonschema:"only these levels: debug, info, warn, error"`
	Q         string   `json:"q,omitempty" jsonschema:"full-text search on the message"`
	Source    string   `json:"source,omitempty" jsonschema:"server or browser"`
	RequestID string   `json:"request_id,omitempty" jsonschema:"only this meta.request_id"`
}

// queryRef names one saved query.
type queryRef struct {
	QueryID int64 `json:"query_id" jsonschema:"a saved query id from journal_list_saved_queries"`
}

// alertsInput is journal_list_alerts's input, which takes nothing.
type alertsInput struct{}

// createAlertInput is journal_create_alert's input.
type createAlertInput struct {
	Name          string `json:"name" jsonschema:"what the rule watches, such as Checkout error burst"`
	SavedQueryID  int64  `json:"saved_query_id" jsonschema:"the saved query to watch, from journal_list_saved_queries"`
	Threshold     int    `json:"threshold" jsonschema:"fire at this many matching entries or more, at least 1"`
	WindowMinutes int    `json:"window_minutes" jsonschema:"within this many minutes, 1 to 1440"`
	Provider      string `json:"provider,omitempty" jsonschema:"antenne (default) for the suite's alert bus, or webhook"`
	WebhookURL    string `json:"webhook_url,omitempty" jsonschema:"for provider webhook: the public http or https URL to post to"`
}

// setAlertInput is journal_set_alert_enabled's input.
type setAlertInput struct {
	AlertID int64 `json:"alert_id" jsonschema:"an alert rule id from journal_list_alerts"`
	Enabled bool  `json:"enabled" jsonschema:"true to turn the rule on, false to pause it"`
}

// alertRef names one alert rule.
type alertRef struct {
	AlertID int64 `json:"alert_id" jsonschema:"an alert rule id from journal_list_alerts"`
}
