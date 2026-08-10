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

// BrowserRequest is what the JavaScript SDK posts. It carries no app: the app
// comes from the key, because the payload is written by code anyone can edit.
type BrowserRequest struct {
	Release     string         `json:"release"`
	Environment string         `json:"environment"`
	Events      []BrowserEvent `json:"events"`
}

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
