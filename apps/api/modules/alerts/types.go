package alerts

type CreateRequest struct {
	Name          string `json:"name"`
	SavedQueryID  int64  `json:"saved_query_id"`
	Threshold     int    `json:"threshold"`
	WindowMinutes int    `json:"window_minutes"`
	WebhookURL    string `json:"webhook_url"`
	WebhookHeader string `json:"webhook_header"`
	WebhookSecret string `json:"webhook_secret"`
}

type UpdateRequest struct {
	Enabled *bool `json:"enabled"`
}

type AlertResponse struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	SavedQueryID  int64   `json:"saved_query_id"`
	QueryName     string  `json:"query_name"`
	Threshold     int     `json:"threshold"`
	WindowMinutes int     `json:"window_minutes"`
	WebhookURL    string  `json:"webhook_url"`
	WebhookHeader *string `json:"webhook_header"`
	Enabled       bool    `json:"enabled"`
	LastFiredAt   *string `json:"last_fired_at"`
	CreatedAt     string  `json:"created_at"`
}

type ListResponse struct {
	Alerts []AlertResponse `json:"alerts"`
}

type AlertEnvelope struct {
	Alert AlertResponse `json:"alert"`
}
