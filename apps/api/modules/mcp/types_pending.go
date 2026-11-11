package mcp

import (
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// PendingItem is one action waiting on the confirm page, with the saved
// query or the alert rule it would delete, as it is now.
type PendingItem struct {
	mcpkit.PendingItem
	Query *PendingQuery `json:"query,omitempty"`
	Alert *PendingAlert `json:"alert,omitempty"`
}

// PendingQuery is the saved query a deletion targets.
type PendingQuery struct {
	ID     int64    `json:"id"`
	Name   string   `json:"name"`
	App    string   `json:"app,omitempty"`
	Levels []string `json:"levels,omitempty"`
	Q      string   `json:"q,omitempty"`
}

// PendingAlert is the alert rule a deletion targets.
type PendingAlert struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	SavedQuery    string `json:"saved_query"`
	Threshold     int    `json:"threshold"`
	WindowMinutes int    `json:"window_minutes"`
	Provider      string `json:"provider"`
	Enabled       bool   `json:"enabled"`
}
