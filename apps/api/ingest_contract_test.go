package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/modules/apikeys"
	"github.com/FacileStudio/Journal/apps/api/schemas"
)

// ingestRequest is one call the suite's shippers make, and the exact answer
// they rely on.
type ingestRequest struct {
	name     string
	path     string
	auth     string
	origin   string
	body     string
	gzipped  bool
	wantCode int
	wantBody string
}

// ingestKeys is one server key, one browser key and one session, the three
// credentials the ingest routes tell apart.
type ingestKeys struct {
	secret, public, session string
}

// TestIngestContractIsFrozen pins what every app that ships to Journal sees:
// the paths, the two key kinds and the legacy token, the accepted payloads and
// the answers, body included. Journal's own deploy must not change any of it.
func TestIngestContractIsFrozen(t *testing.T) {
	app := newLiveApp(t)
	keys := contractKeys(t, app)
	for _, tc := range ingestContract(keys) {
		recorder := sendIngest(t, app, tc)
		got := strings.TrimSpace(recorder.Body.String())
		if recorder.Code != tc.wantCode || got != tc.wantBody {
			t.Errorf("%s: %d %s, want %d %s", tc.name, recorder.Code, got, tc.wantCode, tc.wantBody)
		}
	}
	assertBrowserRateLimit(t, app, keys.public)
}

func contractKeys(t *testing.T, app liveApp) ingestKeys {
	t.Helper()
	service := apikeys.NewService(app.db)
	_, secret, err := service.Create(context.Background(), apikeys.NewKey{App: "nuage"})
	if err != nil {
		t.Fatalf("create a server key: %v", err)
	}
	_, public, err := service.Create(context.Background(), apikeys.NewKey{
		App: "nuage", Kind: schemas.KeyKindPublic, AllowedOrigins: []string{"https://nuage.facile.studio"}, DailyQuota: 1000,
	})
	if err != nil {
		t.Fatalf("create a browser key: %v", err)
	}
	_, session := app.signIn(t, "admin@facile.studio", true)
	return ingestKeys{secret: secret, public: public, session: session}
}

func ingestContract(keys ingestKeys) []ingestRequest {
	const origin = "https://nuage.facile.studio"
	single := `{"app":"nuage","level":"error","message":"upload failed","meta":{"file_id":42}}`
	browserBody := `{"release":"r1","session_id":"tab-1","events":[{"level":"error","message":"boom"}]}`
	return []ingestRequest{
		{"single entry, server key", "/api/ingest", "Bearer " + keys.secret, "", single, false, 201, `{"ingested":1}`},
		{"app filled from the key", "/api/ingest", "Bearer " + keys.secret, "", `{"message":"no app"}`, false, 201, `{"ingested":1}`},
		{"batch", "/api/ingest", "Bearer " + keys.secret, "", `{"entries":[{"message":"a"},{"message":"b","level":"warn"}]}`, false, 201, `{"ingested":2}`},
		{"empty batch", "/api/ingest", "Bearer " + keys.secret, "", `{"entries":[]}`, false, 201, `{"ingested":0}`},
		{"gzip body", "/api/ingest", "Bearer " + keys.secret, "", single, true, 201, `{"ingested":1}`},
		{"legacy token", "/api/ingest", "Bearer " + testIngestToken, "", single, false, 201, `{"ingested":1}`},
		{"another app's entry", "/api/ingest", "Bearer " + keys.secret, "", `{"app":"opus","message":"x"}`, false, 400, `{"error":{"code":"invalid_argument","message":"app \"opus\" is not allowed by this API key (scoped to \"nuage\")"}}`},
		{"no credential", "/api/ingest", "", "", single, false, 401, `{"error":{"code":"unauthenticated","message":"missing bearer token"}}`},
		{"browser key on the server route", "/api/ingest", "Bearer " + keys.public, "", single, false, 401, `{"error":{"code":"unauthenticated","message":"invalid ingest token"}}`},
		{"dashboard session", "/api/ingest", "Bearer " + keys.session, "", single, false, 401, `{"error":{"code":"unauthenticated","message":"invalid ingest token"}}`},
		{"browser, key in the query", "/api/ingest/browser?key=" + keys.public, "", origin, browserBody, false, 201, `{"ingested":1}`},
		{"browser, key as bearer", "/api/ingest/browser", "Bearer " + keys.public, origin, browserBody, false, 201, `{"ingested":1}`},
		{"browser, server key", "/api/ingest/browser?key=" + keys.secret, "", origin, browserBody, false, 401, `{"error":{"code":"unauthenticated","message":"invalid ingest token"}}`},
		{"browser, no key", "/api/ingest/browser", "", origin, browserBody, false, 401, `{"error":{"code":"unauthenticated","message":"missing key"}}`},
		{"browser, no origin", "/api/ingest/browser?key=" + keys.public, "", "", browserBody, false, 403, `{"error":{"code":"permission_denied","message":"this endpoint requires an Origin header"}}`},
		{"browser, unlisted origin", "/api/ingest/browser?key=" + keys.public, "", "https://evil.example", browserBody, false, 403, `{"error":{"code":"permission_denied","message":"origin https://evil.example is not allowed by this key"}}`},
		{"source map list, server key", "/api/sourcemaps?release=r1", "Bearer " + keys.secret, "", "", false, 200, `{"release":"r1","files":[]}`},
		{"source map list, legacy token", "/api/sourcemaps?release=r1", "Bearer " + testIngestToken, "", "", false, 403, `{"error":{"code":"permission_denied","message":"uploading source maps needs a per-app API key, not the shared ingest token"}}`},
	}
}

func sendIngest(t *testing.T, app liveApp, tc ingestRequest) *httptest.ResponseRecorder {
	t.Helper()
	method := http.MethodPost
	if tc.body == "" {
		method = http.MethodGet
	}
	payload := []byte(tc.body)
	if tc.gzipped {
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		_, _ = writer.Write(payload)
		_ = writer.Close()
		payload = compressed.Bytes()
	}
	request := httptest.NewRequest(method, tc.path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	if strings.HasPrefix(tc.path, "/api/ingest/browser") {
		request.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	}
	if tc.gzipped {
		request.Header.Set("Content-Encoding", "gzip")
	}
	if tc.auth != "" {
		request.Header.Set("Authorization", tc.auth)
	}
	if tc.origin != "" {
		request.Header.Set("Origin", tc.origin)
	}
	recorder := httptest.NewRecorder()
	app.router.ServeHTTP(recorder, request)
	return recorder
}

// assertBrowserRateLimit sends one browser past its 60 a minute: the refusal
// is the tronc envelope with a Retry-After, which is what the SDK backs off on.
func assertBrowserRateLimit(t *testing.T, app liveApp, public string) {
	t.Helper()
	tc := ingestRequest{path: "/api/ingest/browser?key=" + public, origin: "https://nuage.facile.studio",
		body: `{"events":[{"message":"loop"}]}`}
	last := sendIngest(t, app, tc)
	for attempt := 0; attempt < 200 && last.Code != http.StatusTooManyRequests; attempt++ {
		last = sendIngest(t, app, tc)
	}
	want := `{"error":{"code":"rate_limited","message":"rate limit exceeded"}}`
	if got := strings.TrimSpace(last.Body.String()); last.Code != http.StatusTooManyRequests || got != want {
		t.Fatalf("a browser past 60 a minute = %d %s, want 429 %s", last.Code, got, want)
	}
	if last.Header().Get("Retry-After") == "" {
		t.Fatal("a 429 without Retry-After leaves the SDK guessing")
	}
}
