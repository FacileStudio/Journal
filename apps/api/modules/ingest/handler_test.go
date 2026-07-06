package ingest

import (
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
