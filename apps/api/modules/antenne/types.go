package antenne

// PoolSettings are the Antenne connection details as the API shows them. The
// secret never leaves the server: SecretSet says whether one is stored.
type PoolSettings struct {
	URL       string `json:"url"`
	SecretSet bool   `json:"secret_set"`
	Enabled   bool   `json:"enabled"`
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

// UpdatePoolRequest is the body of PUT /api/settings/antenne. A missing or
// null antenne_secret keeps the stored secret.
type UpdatePoolRequest struct {
	URL     string  `json:"antenne_url"`
	Secret  *string `json:"antenne_secret"`
	Enabled bool    `json:"antenne_enabled"`
}

// UpdatePoolEventsRequest is the body of PUT /api/settings/antenne/events.
type UpdatePoolEventsRequest struct {
	Events []PoolEventToggle `json:"events"`
}
