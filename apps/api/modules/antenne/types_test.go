package antenne

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSettingsResponseJSONShape(t *testing.T) {
	body, err := json.Marshal(PoolSettingsResponse{
		Settings:     PoolSettings{URL: "https://antenne.example", SecretSet: true, Enabled: true},
		Connected:    true,
		FromEnv:      true,
		ConnectError: "boom",
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"settings":      map[string]any{"url": "https://antenne.example", "secret_set": true, "enabled": true},
		"connected":     true,
		"from_env":      true,
		"connect_error": "boom",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response shape = %v, want %v", got, want)
	}
}

func TestUpdateRequestJSONShape(t *testing.T) {
	var req UpdatePoolRequest
	if err := json.Unmarshal([]byte(`{"antenne_url":"u","antenne_secret":"s","antenne_enabled":true}`), &req); err != nil {
		t.Fatal(err)
	}
	if req.URL != "u" || req.Secret == nil || *req.Secret != "s" || !req.Enabled {
		t.Fatalf("request = %+v", req)
	}
	var keep UpdatePoolRequest
	if err := json.Unmarshal([]byte(`{"antenne_url":"u","antenne_enabled":true}`), &keep); err != nil {
		t.Fatal(err)
	}
	if keep.Secret != nil {
		t.Fatal("a body without antenne_secret must keep the stored secret")
	}
}
