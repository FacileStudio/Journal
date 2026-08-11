package journal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeMap(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(`{"version":3,"sources":["a.js"],"mappings":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
}

// The uploader runs at every boot, so the thing that matters is that a restart
// costs one request and not a full re-upload of every map in the build.
func TestUploadSourceMapsSkipsWhatTheServerHolds(t *testing.T) {
	dir := t.TempDir()
	writeMap(t, dir, "already.js.map")
	writeMap(t, dir, "fresh.js.map")

	var uploaded []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			t.Errorf("missing credential on %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodGet:
			if got := r.URL.Query().Get("release"); got != "v1" {
				t.Errorf("release = %q, want v1", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"files": []string{"already.js"}})
		case http.MethodPost:
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			uploaded = append(uploaded, body["file"])
			_ = json.NewEncoder(w).Encode(map[string]bool{"stored": true})
		}
	}))
	defer server.Close()

	result, err := UploadSourceMaps(context.Background(), Config{URL: server.URL, Token: "key"}, dir, "v1")
	if err != nil {
		t.Fatalf("UploadSourceMaps: %v", err)
	}

	if result.Found != 2 || result.Uploaded != 1 || result.Skipped != 1 {
		t.Fatalf("result = %+v, want 2 found / 1 uploaded / 1 skipped", result)
	}
	if len(uploaded) != 1 || uploaded[0] != "fresh.js" {
		t.Fatalf("uploaded %v, want only fresh.js — and keyed on the bundle name, not the .map", uploaded)
	}
}

// Most apps ship no maps. That is the normal case and must not be an error, or
// every app on the SDK starts logging a failure it cannot act on.
func TestUploadSourceMapsIsQuietWithoutADirectory(t *testing.T) {
	result, err := UploadSourceMaps(context.Background(), Config{URL: "http://unused", Token: "k"}, filepath.Join(t.TempDir(), "absent"), "v1")
	if err != nil {
		t.Fatalf("a missing directory was an error: %v", err)
	}
	if result.Found != 0 {
		t.Fatalf("result = %+v, want nothing found", result)
	}
}

func TestUploadSourceMapsNeedsARelease(t *testing.T) {
	if _, err := UploadSourceMaps(context.Background(), Config{URL: "http://x", Token: "k"}, t.TempDir(), "  "); err == nil {
		t.Fatal("a blank release was accepted; maps would be stored under a name nothing reports")
	}
}

// A refusal has to surface: silently reporting success would leave somebody
// wondering why traces never resolve.
func TestUploadSourceMapsReportsARejection(t *testing.T) {
	dir := t.TempDir()
	writeMap(t, dir, "a.js.map")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{"files": []string{}})
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"needs a per-app API key"}}`))
	}))
	defer server.Close()

	if _, err := UploadSourceMaps(context.Background(), Config{URL: server.URL, Token: "k"}, dir, "v1"); err == nil {
		t.Fatal("a 403 was reported as success")
	}
}
