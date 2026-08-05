package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/go-chi/chi/v5"
)

// testRouter builds the real router with a nil database. Nothing here serves a
// request that would touch it: the tests walk the route tree and read the
// reference, both of which are assembled before any handler runs.
func testRouter() chi.Router {
	return buildRouter(nil, env.Config{}, slog.New(slog.DiscardHandler))
}

// TestEveryRouteIsDocumented is the reason the registry can be trusted: a route
// added to any module's RegisterRoutes without a matching entry fails here.
func TestEveryRouteIsDocumented(t *testing.T) {
	if missing := apiref.Undocumented(testRouter(), referenceConfig()); len(missing) > 0 {
		t.Errorf("routes missing from the API registry: %v", missing)
	}
}

func TestReferenceIsServed(t *testing.T) {
	router := testRouter()

	page := httptest.NewRecorder()
	router.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("GET /docs = %d, want 200", page.Code)
	}

	spec := httptest.NewRecorder()
	router.ServeHTTP(spec, httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil))
	if spec.Code != http.StatusOK {
		t.Fatalf("GET /docs/openapi.json = %d, want 200", spec.Code)
	}

	body, err := io.ReadAll(spec.Body)
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var document struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("spec is not JSON: %v", err)
	}
	if document.OpenAPI != "3.1.0" {
		t.Errorf("openapi = %q, want 3.1.0", document.OpenAPI)
	}
	if _, ok := document.Paths["/ingest"]["post"]; !ok {
		t.Errorf("spec does not describe POST /ingest")
	}
}
