package alerts

// CreateRequest is the body of an alert definition.
type CreateRequest struct {
	Name          string `json:"name"`
	SavedQueryID  int64  `json:"saved_query_id"`
	Provider      string `json:"provider,omitempty"` // webhook or antenne
	Threshold     int    `json:"threshold"`
	WindowMinutes int    `json:"window_minutes"`
	WebhookURL    string `json:"webhook_url"`
	WebhookHeader string `json:"webhook_header"`
	WebhookSecret string `json:"webhook_secret"`
}

// UpdateRequest is the body of an alert update — currently just enable/disable.
type UpdateRequest struct {
	Enabled *bool `json:"enabled"`
}

// AlertResponse describes one alert for the client.
type AlertResponse struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	SavedQueryID  int64   `json:"saved_query_id"`
	Provider      string  `json:"provider"` // webhook or antenne
	QueryName     string  `json:"query_name"`
	Threshold     int     `json:"threshold"`
	WindowMinutes int     `json:"window_minutes"`
	WebhookURL    string  `json:"webhook_url"`
	WebhookHeader *string `json:"webhook_header"`
	Enabled       bool    `json:"enabled"`
	LastFiredAt   *string `json:"last_fired_at"`
	CreatedAt     string  `json:"created_at"`
}

// ListResponse is the list of every alert.
type ListResponse struct {
	Alerts []AlertResponse `json:"alerts"`
}

// AlertEnvelope wraps the fired alert, what the webhook posts.
type AlertEnvelope struct {
	Alert AlertResponse `json:"alert"`
}