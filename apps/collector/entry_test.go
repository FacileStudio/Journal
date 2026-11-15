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
		{"plain stderr", "something broke", streamStderr, "warn"},
		{"postgres LOG on stderr", "2026-11-14 10:00:00.123 UTC [1] LOG:  checkpoint complete", streamStderr, "info"},
		{"postgres bare LOG", "LOG:  database system is ready", streamStderr, "info"},
		{"postgres ERROR", "ERROR:  relation \"x\" does not exist", streamStderr, "error"},
		{"postgres FATAL", "2026-11-14 10:00:00 UTC [4] FATAL:  terminating connection", streamStderr, "error"},
		{"postgres PANIC", "PANIC:  could not write", streamStdout, "error"},
		{"postgres WARNING", "WARNING:  no privileges", streamStderr, "warn"},
		{"postgres NOTICE", "NOTICE:  table skipped", streamStderr, "info"},
		{"logfmt level", `ts=1 level=error msg="boom"`, streamStdout, "error"},
		{"logfmt quoted warn", `level="warn" msg=x`, streamStderr, "warn"},
		{"bracketed ERROR", "[ERROR] connection refused", streamStdout, "error"},
		{"bracketed with time", "2026-11-14T10:00:00Z [warn] slow query", streamStdout, "warn"},
		{"word error mid line stays plain", "no error here: ok", streamStdout, "info"},
		{"json level error on stdout", `{"level":"error","msg":"boom"}`, streamStdout, "error"},
		{"json level warn", `{"level":"warn"}`, streamStdout, "warn"},
		{"json warning normalized", `{"level":"WARNING"}`, streamStdout, "warn"},
		{"json lvl field", `{"lvl":"debug"}`, streamStderr, "debug"},
		{"json severity field", `{"severity":"fatal"}`, streamStdout, "error"},
		{"json unknown level", `{"level":"verbose"}`, streamStdout, "info"},
		{"json numeric level ignored", `{"level":30}`, streamStderr, "warn"},
		{"json without level stderr", `{"msg":"no level"}`, streamStderr, "warn"},
		{"json without level stdout", `{"msg":"no level"}`, streamStdout, "info"},
		{"invalid json stderr", `{not json`, streamStderr, "warn"},
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
	if e.Level != "warn" {
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

func TestCollectMinLevel(t *testing.T) {
	mins := parseMinLevels("labparasite-apppupille=warn, journal-api=warn,bad=loud,=info,noeq")
	if len(mins) != 2 {
		t.Fatalf("mins = %v", mins)
	}
	cases := []struct {
		app, level string
		drop       bool
	}{
		{"journal-api", "info", true},
		{"journal-api", "warn", false},
		{"journal-api", "error", false},
		{"labparasite-apppupille-6bypjx-api-1", "info", true},
		{"labparasite-apppupille-6bypjx-api-1", "error", false},
		{"labparasite-apppupillex", "info", false},
		{"sablier", "debug", false},
	}
	for _, tc := range cases {
		if got := belowMinLevel(mins, tc.app, tc.level); got != tc.drop {
			t.Fatalf("belowMinLevel(%s, %s) = %v, want %v", tc.app, tc.level, got, tc.drop)
		}
	}
	if belowMinLevel(parseMinLevels(""), "x", "debug") {
		t.Fatal("empty config dropped a line")
	}
}

func TestKeepEntryDropsBlankLines(t *testing.T) {
	for _, message := range []string{"", "   ", "\t"} {
		if keepEntry(filters{}, entry{App: "a", Level: "info", Message: message}) {
			t.Fatalf("kept blank message %q, the API rejects the whole batch with 400", message)
		}
	}
	if !keepEntry(filters{}, entry{App: "a", Level: "info", Message: "x"}) {
		t.Fatal("dropped a real line")
	}
}

func TestCollectSDKAppsDropsOnlyShippedSlogLines(t *testing.T) {
	f := filters{sdkApps: parseAppList(" facile-sablier-lbgdwy-api , ,vision")}
	slog := `{"time":"2026-11-15T19:59:32Z","level":"INFO","msg":"Reminder sent","sent":0}`
	cases := []struct {
		app, message string
		keep         bool
	}{
		{"facile-sablier-lbgdwy-api-19", slog, false},
		{"vision", slog, false},
		{"facile-sablier-lbgdwy-api-19", "panic: runtime error: index out of range", true},
		{"facile-sablier-lbgdwy-api-19", `{"event":"boot"}`, true},
		{"facile-sablier-lbgdwy-db-1", slog, true},
		{"visionary", slog, true},
	}
	for _, tc := range cases {
		if got := keepEntry(f, entry{App: tc.app, Level: "info", Message: tc.message}); got != tc.keep {
			t.Fatalf("keepEntry(%s, %q) = %v, want %v", tc.app, tc.message, got, tc.keep)
		}
	}
}
