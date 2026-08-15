package ingest

// IngestEntry is one log line to be stored.
type IngestEntry struct {
	App     string         `json:"app"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Ts      string         `json:"ts"`
	Meta    map[string]any `json:"meta"`
}

// IngestRequest is the body of the server-to-server ingest endpoint.
type IngestRequest struct {
	Entries []IngestEntry  `json:"entries"`
	App     string         `json:"app"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Ts      string         `json:"ts"`
	Meta    map[string]any `json:"meta"`
}

// IngestResponse reports how many entries were actually stored.
type IngestResponse struct {
	Ingested int `json:"ingested"`
}

// BrowserRequest is what the JavaScript SDK posts. It carries no app: the app
// comes from the key, because the payload is written by code anyone can edit.
type BrowserRequest struct {
	Release     string         `json:"release"`
	Environment string         `json:"environment"`
	Events      []BrowserEvent `json:"events"`
}

// BrowserEvent is one event posted by the JavaScript SDK.
type BrowserEvent struct {
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Ts      string         `json:"ts"`
	Kind    string         `json:"kind"`
	Stack   string         `json:"stack"`
	URL     string         `json:"url"`
	Route   string         `json:"route"`
	Count   int            `json:"count"`
	User    BrowserUser    `json:"user"`
	Meta    map[string]any `json:"meta"`
}

// BrowserUser keys on email, which is how the suite identifies a person across
// apps until the shared user sync lands.
type BrowserUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}
