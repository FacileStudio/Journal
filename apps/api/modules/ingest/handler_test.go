package ingest

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestClampTimestamp(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		parsed time.Time
		want   time.Time
	}{
		{"past stays", now.Add(-48 * time.Hour), now.Add(-48 * time.Hour)},
		{"now stays", now, now},
		{"slightly future stays", now.Add(4 * time.Minute), now.Add(4 * time.Minute)},
		{"boundary stays", now.Add(5 * time.Minute), now.Add(5 * time.Minute)},
		{"past boundary clamps", now.Add(5*time.Minute + time.Second), now},
		{"year 9999 clamps", time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC), now},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clampTimestamp(tc.parsed, now); !got.Equal(tc.want) {
				t.Fatalf("clampTimestamp(%v) = %v, want %v", tc.parsed, got, tc.want)
			}
		})
	}
}

func batchBody(t *testing.T, size int) []byte {
	t.Helper()
	entries := make([]IngestEntry, size)
	for i := range entries {
		entries[i] = IngestEntry{App: "nuage", Message: "boom"}
	}
	body, err := json.Marshal(IngestRequest{Entries: entries})
	if err != nil {
		t.Fatalf("marshal batch: %v", err)
	}
	return body
}

func TestBatchCap(t *testing.T) {
	handler := newHandler(nil)

	request := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(batchBody(t, maxBatchEntries+1)))
	recorder := httptest.NewRecorder()
	handler.ingest(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "1000") {
		t.Fatalf("error body %q does not name the limit", recorder.Body.String())
	}
}

func TestBatchCapGzip(t *testing.T) {
	handler := newHandler(nil)

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(batchBody(t, maxBatchEntries+1)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/ingest", &compressed)
	request.Header.Set("Content-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	handler.ingest(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "1000") {
		t.Fatalf("error body %q does not name the limit", recorder.Body.String())
	}
}

func TestCapMessage(t *testing.T) {
	cases := []struct {
		name    string
		message string
		capped  bool
	}{
		{"short unchanged", "hello", false},
		{"exactly max unchanged", strings.Repeat("a", maxMessageBytes), false},
		{"over max truncated", strings.Repeat("a", maxMessageBytes+1), true},
		{"way over truncated", strings.Repeat("x", maxMessageBytes*3), true},
		{"multibyte boundary safe", strings.Repeat("é", maxMessageBytes/2+10), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := capMessage(tc.message)
			if !tc.capped {
				if got != tc.message {
					t.Fatal("message under cap was modified")
				}
				return
			}
			if !strings.HasSuffix(got, truncationSuffix) {
				t.Fatal("truncated message lacks suffix")
			}
			if len(got) > maxMessageBytes+len(truncationSuffix) {
				t.Fatalf("truncated message too long: %d bytes", len(got))
			}
			if !utf8.ValidString(got) {
				t.Fatal("truncation produced invalid UTF-8")
			}
			if !strings.HasPrefix(tc.message, strings.TrimSuffix(got, truncationSuffix)) {
				t.Fatal("truncated content is not a prefix of the original")
			}
		})
	}
}
