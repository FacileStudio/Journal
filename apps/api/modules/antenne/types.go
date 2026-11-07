package antenne

// PoolSettings are the Antenne connection details stored in AppSetting.
type PoolSettings struct {
	URL     string `json:"url"`
	Secret  string `json:"secret"`
	Enabled bool   `json:"enabled"`
}

// PoolSettingsResponse is what the HTTP handler returns.
type PoolSettingsResponse struct {
	Settings     PoolSettings `json:"settings"`
	Connected    bool         `json:"connected"`
	FromEnv      bool         `json:"from_env"`
	ConnectError string       `json:"connect_error,omitempty"`
}

// PoolEventToggle is a single enable/disable switch for a pooled event type.
type PoolEventToggle struct {
	Event   string `json:"event"`
	Enabled bool   `json:"enabled"`
}

// PoolEventsResponse is the list of toggles for the dashboard.
type PoolEventsResponse struct {
	Events []PoolEventToggle `json:"events"`
}

// UpdatePoolRequest is the body of PUT /api/settings/antenne.
type UpdatePoolRequest struct {
	URL     string `json:"antenne_url"`
	Secret  string `json:"antenne_secret"`
	Enabled bool   `json:"antenne_enabled"`
}

// UpdatePoolEventsRequest is the body of PUT /api/settings/antenne/events.
type UpdatePoolEventsRequest struct {
	Events []PoolEventToggle `json:"events"`
}
