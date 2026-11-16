package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	if len(scripts) == 0 || slices.ContainsFunc(scripts, func(s string) bool { return strings.HasPrefix(s, "https:") }) {
		t.Fatalf("script-src = %v, want inline script hashes and no remote origin", scripts)
	}
}
