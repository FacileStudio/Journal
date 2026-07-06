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

type Cursor struct {
	Ts string `json:"ts"`
	ID int64  `json:"id"`
}

type ListResponse struct {
	Entries    []LogResponse `json:"entries"`
	NextBefore *Cursor       `json:"next_before"`
}

type AppSummary struct {
	Name     string `json:"name"`
	Count    int64  `json:"count"`
	LastSeen string `json:"last_seen"`
}

type AppsResponse struct {
	Apps []AppSummary `json:"apps"`
}

type HistogramBucket struct {
	Ts     string           `json:"ts"`
	Counts map[string]int64 `json:"counts"`
}

type HistogramResponse struct {
	BucketSeconds int64             `json:"bucket_seconds"`
	Buckets       []HistogramBucket `json:"buckets"`
}

type ContextResponse struct {
	Entries  []LogResponse `json:"entries"`
	AnchorID int64         `json:"anchor_id"`
}
