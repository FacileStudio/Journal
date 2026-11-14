package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestSecurityConfigOpensWhatTheLandingFetchesAndLoads(t *testing.T) {
	dir := t.TempDir()
	page := `<!doctype html><html><head><script>window.a=1</script></head><body></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := securityConfig(dir, slog.Default())
	if !slices.Contains(cfg.CSPSources["connect-src"], ardoiseCatalogOrigin) {
		t.Fatalf("connect-src = %v, want the Ardoise catalog origin", cfg.CSPSources["connect-src"])
	}
	scripts := cfg.CSPSources["script-src"]
	if !slices.Contains(scripts, visionOrigin) || len(scripts) < 2 {
		t.Fatalf("script-src = %v, want the inline script hash and the Vision origin", scripts)
	}
}
