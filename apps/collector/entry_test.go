package main

import (
	"strings"
	"testing"
	"time"
)

func TestDetectLevel(t *testing.T) {
	cases := []struct {
		name    string
		message string
		stream  byte
		want    string
	}{
		{"plain stdout", "hello world", streamStdout, "info"},
		{"plain stderr", "something broke", streamStderr, "error"},
		{"json level error on stdout", `{"level":"error","msg":"boom"}`, streamStdout, "error"},
		{"json level warn", `{"level":"warn"}`, streamStdout, "warn"},
		{"json warning normalized", `{"level":"WARNING"}`, streamStdout, "warn"},
		{"json lvl field", `{"lvl":"debug"}`, streamStderr, "debug"},
		{"json severity field", `{"severity":"fatal"}`, streamStdout, "error"},
		{"json unknown level", `{"level":"verbose"}`, streamStdout, "info"},
		{"json numeric level ignored", `{"level":30}`, streamStderr, "error"},
		{"json without level stderr", `{"msg":"no level"}`, streamStderr, "error"},
		{"json without level stdout", `{"msg":"no level"}`, streamStdout, "info"},
		{"invalid json stderr", `{not json`, streamStderr, "error"},
		{"json with leading space", `  {"level":"debug"}`, streamStdout, "debug"},
		{"trace maps to debug", `{"level":"trace"}`, streamStdout, "debug"},
		{"err maps to error", `{"level":"err"}`, streamStdout, "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectLevel(tc.message, tc.stream); got != tc.want {
				t.Fatalf("detectLevel(%q, %d) = %q, want %q", tc.message, tc.stream, got, tc.want)
			}
		})
	}
}

func TestSplitTimestamp(t *testing.T) {
	ts, msg := splitTimestamp("2026-07-06T10:11:12.123456789Z hello there")
	want := time.Date(2026, 7, 6, 10, 11, 12, 123456789, time.UTC)
	if !ts.Equal(want) {
		t.Fatalf("ts = %v, want %v", ts, want)
	}
	if msg != "hello there" {
		t.Fatalf("msg = %q", msg)
	}

	ts, msg = splitTimestamp("no timestamp here")
	if !ts.IsZero() {
		t.Fatalf("expected zero ts, got %v", ts)
	}
	if msg != "no timestamp here" {
		t.Fatalf("msg = %q", msg)
	}

	ts, msg = splitTimestamp("2026-07-06T10:11:12Z")
	if ts.IsZero() || msg != "" {
		t.Fatalf("timestamp-only line: ts=%v msg=%q", ts, msg)
	}
}

func TestMapLine(t *testing.T) {
	ts := time.Date(2026, 7, 6, 8, 0, 0, 500, time.UTC)
	e := mapLine("nuage", "abcdef1234567890", streamStderr, ts, "disk full\r\n")
	if e.App != "nuage" {
		t.Fatalf("app = %q", e.App)
	}
	if e.Level != "error" {
		t.Fatalf("level = %q", e.Level)
	}
	if e.Message != "disk full" {
		t.Fatalf("message = %q", e.Message)
	}
	if e.TS != ts.Format(time.RFC3339Nano) {
		t.Fatalf("ts = %q", e.TS)
	}
	if e.Meta["container_id"] != "abcdef123456" {
		t.Fatalf("container_id = %q", e.Meta["container_id"])
	}
	if e.Meta["stream"] != "stderr" {
		t.Fatalf("stream = %q", e.Meta["stream"])
	}

	e = mapLine("app", "short", streamStdout, time.Time{}, "ok")
	if e.TS != "" {
		t.Fatalf("expected empty ts, got %q", e.TS)
	}
	if e.Meta["container_id"] != "short" {
		t.Fatalf("container_id = %q", e.Meta["container_id"])
	}
	if e.Meta["stream"] != "stdout" || e.Level != "info" {
		t.Fatalf("stream=%q level=%q", e.Meta["stream"], e.Level)
	}

	huge := strings.Repeat("a", maxLineBytes+100)
	e = mapLine("app", "short", streamStdout, time.Time{}, huge)
	if len(e.Message) != maxLineBytes {
		t.Fatalf("message not capped: %d", len(e.Message))
	}
}

func TestSkipContainer(t *testing.T) {
	self := containerSummary{ID: "aabbccddeeff00112233", Labels: map[string]string{}}
	if !skipContainer(self, "aabbccddeeff") {
		t.Fatal("must skip itself by hostname prefix")
	}
	ignored := containerSummary{ID: "0123456789ab", Labels: map[string]string{"journal.ignore": "true"}}
	if !skipContainer(ignored, "ffffffffffff") {
		t.Fatal("must skip journal.ignore=true")
	}
	normal := containerSummary{ID: "0123456789ab", Labels: map[string]string{}}
	if skipContainer(normal, "ffffffffffff") {
		t.Fatal("must not skip normal container")
	}
}

func TestAppName(t *testing.T) {
	labeled := containerSummary{ID: "0123456789abcdef", Names: []string{"/my-container"}, Labels: map[string]string{"journal.app": "custom"}}
	if got := appName(labeled); got != "custom" {
		t.Fatalf("appName = %q", got)
	}
	named := containerSummary{ID: "0123456789abcdef", Names: []string{"/my-container"}, Labels: map[string]string{}}
	if got := appName(named); got != "my-container" {
		t.Fatalf("appName = %q", got)
	}
	anonymous := containerSummary{ID: "0123456789abcdef", Labels: map[string]string{}}
	if got := appName(anonymous); got != "0123456789ab" {
		t.Fatalf("appName = %q", got)
	}
}
