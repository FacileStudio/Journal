package ingest

type IngestEntry struct {
	App     string         `json:"app"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Ts      string         `json:"ts"`
	Meta    map[string]any `json:"meta"`
}

type IngestRequest struct {
	Entries []IngestEntry  `json:"entries"`
	App     string         `json:"app"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Ts      string         `json:"ts"`
	Meta    map[string]any `json:"meta"`
}

type IngestResponse struct {
	Ingested int `json:"ingested"`
}
