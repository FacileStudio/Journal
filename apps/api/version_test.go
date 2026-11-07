package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestVersionIsTheReleaseVersion(t *testing.T) {
	if version == "dev" {
		t.Skip("version is set by -ldflags at build time")
	}
	raw, err := os.ReadFile("../client/package.json")
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	versionHandler(recorder, httptest.NewRequest(http.MethodGet, "/version", nil))
	want := `{"version":"v` + pkg.Version + `"}`
	if strings.TrimSpace(recorder.Body.String()) != want {
		t.Errorf("GET /version = %q, want %q", recorder.Body.String(), want)
	}
}
