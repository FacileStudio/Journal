package sourcemaps

// UploadRequest carries one map. There is no app field: the key's app is
// authoritative, exactly as on /ingest.
type UploadRequest struct {
	Release string `json:"release"`
	File    string `json:"file"`
	Map     string `json:"map"`
}

// UploadResponse reports whether the upload stored anything, i.e. the map was
// not already held.
type UploadResponse struct {
	Stored bool `json:"stored"`
}

// ListResponse is what an uploader reads before sending anything, so a restart
// re-uploads nothing.
type ListResponse struct {
	Release string   `json:"release"`
	Files   []string `json:"files"`
}

// ReleaseSummary is one release's worth of maps, for the dashboard.
type ReleaseSummary struct {
	App       string `json:"app"`
	Release   string `json:"release"`
	Files     int    `json:"files"`
	Bytes     int64  `json:"bytes"`
	CreatedAt string `json:"created_at"`
}

// ReleasesResponse lists every release with stored maps, for the dashboard.
type ReleasesResponse struct {
	Releases []ReleaseSummary `json:"releases"`
}
