package mcp

import (
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// list is the curated tool set. Log reads are open world: apps and visitors'
// browsers write the messages and meta. The two deletions are confirmed.
func (t *toolset) list() []mcpkit.Tool {
	return append(append(append(t.logReads(), t.queryTools()...), t.alertTools()...), t.sensitive()...)
}

// logReads are the four tools over log entries.
func (t *toolset) logReads() []mcpkit.Tool {
	return []mcpkit.Tool{
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolSearch, Title: "Search logs",
			Description: "Find log entries, newest first, across every app or one. Filter by app, level, q (full-text " +
				"words, \"quoted phrase\", -excluded), source (server or browser), request_id, session_id and a time " +
				"window. since and until take an RFC 3339 time or a duration back from now such as 15m, 6h or 7d; the " +
				"default window is the last 24 hours. Messages are cut to 1,000 characters; response_format detailed " +
				"adds meta. Pass next_cursor back as cursor for older entries. Read one entry in full with journal_read_log.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true, OpenWorld: true},
		}, t.search),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolRead, Title: "Read a log entry",
			Description: "Read one log entry in full: message, meta, request and session ids. Pass before and after " +
				"(up to 50 each) for the entries logged just before and after it across all apps, and resolve_stack to " +
				"map a browser stack trace through the uploaded source maps.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true, OpenWorld: true},
		}, t.read),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolCount, Title: "Count logs over time",
			Description: "Count log entries per level over a time window, in up to 90 buckets, with the totals. Same " +
				"filters as journal_search_logs. Use it to see when errors started or spiked before searching.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true},
		}, t.count),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolApps, Title: "List apps",
			Description: "List every app that has sent logs, with its entry count and when it last logged, most " +
				"recent first. Use the names as the app filter of the other tools.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true},
		}, t.apps),
	}
}

// queryTools are the saved query tools that read or add.
func (t *toolset) queryTools() []mcpkit.Tool {
	return []mcpkit.Tool{
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolQueries, Title: "List saved queries",
			Description: "List the saved queries everyone in Journal shares: id, name and filters. Run one by passing " +
				"its filters to journal_search_logs.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true},
		}, t.listQueries),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolSaveQuery, Title: "Save a query",
			Description: "Save a named filter (app, levels, q, source, request_id) for everyone in Journal, as the " +
				"Save button of the log explorer does. Names are unique. An alert rule needs a saved query to watch.",
			Annotations: &mcpkit.Annotations{},
		}, t.saveQuery),
	}
}

// alertTools are the administrator tools that read, add or switch alert
// rules.
func (t *toolset) alertTools() []mcpkit.Tool {
	return []mcpkit.Tool{
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolAlerts, Title: "List alert rules",
			Description: "List the alert rules (administrators only): the saved query each watches, the threshold of " +
				"matching entries within window_minutes, where it notifies (Antenne, or a webhook host), whether it is " +
				"on and when it last fired.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true},
		}, t.listAlerts),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolCreateAlert, Title: "Create an alert rule",
			Description: "Create an alert rule (administrators only) that fires when a saved query matches at least " +
				"threshold entries within window_minutes (1 to 1440). provider antenne notifies through the suite's " +
				"Antenne bus; webhook posts to webhook_url, which must be a public http or https URL. It starts enabled.",
			Annotations: &mcpkit.Annotations{OpenWorld: true},
		}, t.createAlert),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolSetAlert, Title: "Turn an alert rule on or off",
			Description: "Enable or disable one alert rule (administrators only). A disabled rule keeps its settings " +
				"and stops firing until it is enabled again.",
			Annotations: &mcpkit.Annotations{Destructive: true, Idempotent: true},
		}, t.setAlert),
	}
}

// sensitive are the tools the user confirms.
func (t *toolset) sensitive() []mcpkit.Tool {
	return []mcpkit.Tool{
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolDeleteQuery, Title: "Delete a saved query",
			Description: "Delete a saved query for everyone, for good. A query an alert rule watches cannot be deleted " +
				"until that rule is. The user always confirms: in your client when it can ask, otherwise the result has " +
				"status pending_confirmation and a confirm_url to give the user. Nothing is deleted before they confirm.",
			Annotations: &mcpkit.Annotations{Destructive: true, Idempotent: true},
		}, t.deleteQuery),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolDeleteAlert, Title: "Delete an alert rule",
			Description: "Delete an alert rule for good (administrators only). The user always confirms: in your client " +
				"when it can ask, otherwise the result has status pending_confirmation and a confirm_url to give the " +
				"user. Nothing is deleted before they confirm. To pause a rule instead, use journal_set_alert_enabled.",
			Annotations: &mcpkit.Annotations{Destructive: true, Idempotent: true},
		}, t.deleteAlert),
	}
}
