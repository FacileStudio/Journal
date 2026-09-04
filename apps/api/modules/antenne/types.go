package antenne

// PoolSettings are the Antenne connection details stored in AppSetting.
type PoolSettings struct {
	URL     string
	Secret  string
	Enabled bool
}

// PoolSettingsResponse is what the HTTP handler returns.
type PoolSettingsResponse struct {
	Settings     PoolSettings
	Connected    bool
	FromEnv      bool
	ConnectError string `json:",omitempty"`
}

// PoolEventToggle is a single enable/disable switch for a pooled event type.
type PoolEventToggle struct {
	Event   string
	Enabled bool
}

// PoolEventsResponse is the list of toggles for the dashboard.
type PoolEventsResponse struct {
	Events []PoolEventToggle
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
