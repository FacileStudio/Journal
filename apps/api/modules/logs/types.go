package logs

type LogResponse struct {
	ID         int64          `json:"id"`
	App        string         `json:"app"`
	Level      string         `json:"level"`
	Message    string         `json:"message"`
	Meta       map[string]any `json:"meta,omitempty"`
	CreatedAt  string         `json:"created_at"`
	ReceivedAt string         `json:"received_at"`
}

type ListResponse struct {
	Entries    []LogResponse `json:"entries"`
	NextBefore *int64        `json:"next_before"`
}

type AppSummary struct {
	Name     string `json:"name"`
	Count    int64  `json:"count"`
	LastSeen string `json:"last_seen"`
}

type AppsResponse struct {
	Apps []AppSummary `json:"apps"`
}
