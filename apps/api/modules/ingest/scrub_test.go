package ingest

import "testing"

// The secret-key path trusted the producer's meta completely, which made
// every server the one place a password could hide forever. These keys are
// scrubbed on the way in, at any depth, and nothing else is touched.
func TestServerMetaScrubbing(t *testing.T) {
	meta := map[string]any{
		"user_id": float64(42),
		"path":    "/login",
		"token":   "abc",
		"nested":  map[string]any{"password": "hunter2", "keep": "yes"},
		"list":    []any{map[string]any{"api_key": "x", "ok": true}, "plain"},
	}

	out := scrubServerMeta(meta)

	if out["user_id"] != float64(42) || out["path"] != "/login" {
		t.Fatalf("benign keys were touched: %v", out)
	}
	if out["token"] != scrubbedValue {
		t.Fatalf("token = %v, want it scrubbed", out["token"])
	}
	nested, ok := out["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested = %T, want a map", out["nested"])
	}
	if nested["password"] != scrubbedValue || nested["keep"] != "yes" {
		t.Fatalf("nested = %v, want password scrubbed and keep intact", nested)
	}
	list, ok := out["list"].([]any)
	if !ok {
		t.Fatalf("list = %T, want a slice", out["list"])
	}
	first, ok := list[0].(map[string]any)
	if !ok || first["api_key"] != scrubbedValue || first["ok"] != true {
		t.Fatalf("list[0] = %v, want api_key scrubbed", list[0])
	}
	if list[1] != "plain" {
		t.Fatalf("list[1] = %v, want it untouched", list[1])
	}
}

func TestServerMetaScrubbingKeepsNilNil(t *testing.T) {
	if out := scrubServerMeta(nil); out != nil {
		t.Fatalf("nil became %v", out)
	}
}
