package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

const levelWords = `(?i:debug[1-5]?|trace|info|notice|log|warn|warning|error|err|fatal|panic)`

var (
	prefixLevel  = regexp.MustCompile(`^(?:\d{4}-\d\d-\d\d[ T][\d:.]+(?:Z| [A-Z]{2,5})?\s+)?(?:\[\d+\]\s+)?(?:\S+@\S+\s+)?(` + levelWords + `):\s`)
	bracketLevel = regexp.MustCompile(`^(?:\d{4}-\d\d-\d\d[ T][\d:.]+(?:Z| [A-Z]{2,5})?\s+)?\[(` + levelWords + `)\]`)
	logfmtLevel  = regexp.MustCompile(`(?:^|\s)(?:level|lvl)=["']?(` + levelWords + `)["']?(?:\s|$)`)
)

var levelRank = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}

type entry struct {
	App     string            `json:"app"`
	Level   string            `json:"level"`
	Message string            `json:"message"`
	TS      string            `json:"ts,omitempty"`
	Meta    map[string]string `json:"meta"`
}

func splitTimestamp(line string) (time.Time, string) {
	i := strings.IndexByte(line, ' ')
	head, rest := line, ""
	if i >= 0 {
		head, rest = line[:i], line[i+1:]
	}
	ts, err := time.Parse(time.RFC3339Nano, head)
	if err != nil {
		return time.Time{}, line
	}
	return ts, rest
}

func normalizeLevel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug", "trace", "debug1", "debug2", "debug3", "debug4", "debug5":
		return "debug"
	case "info", "notice", "log":
		return "info"
	case "warn", "warning":
		return "warn"
	case "error", "err", "fatal", "panic":
		return "error"
	default:
		return "info"
	}
}

func detectLevel(message string, stream byte) string {
	trimmed := strings.TrimSpace(message)
	if strings.HasPrefix(trimmed, "{") {
		var obj map[string]any
		if json.Unmarshal([]byte(trimmed), &obj) == nil {
			for _, key := range []string{"level", "lvl", "severity"} {
				if v, ok := obj[key].(string); ok {
					return normalizeLevel(v)
				}
			}
		}
	}
	for _, re := range []*regexp.Regexp{prefixLevel, bracketLevel, logfmtLevel} {
		if m := re.FindStringSubmatch(trimmed); m != nil {
			return normalizeLevel(m[1])
		}
	}
	if stream == streamStderr {
		return "warn"
	}
	return "info"
}

func parseMinLevels(raw string) map[string]string {
	out := map[string]string{}
	for _, pair := range strings.Split(raw, ",") {
		app, level, ok := strings.Cut(strings.TrimSpace(pair), "=")
		app, level = strings.TrimSpace(app), strings.ToLower(strings.TrimSpace(level))
		if _, known := levelRank[level]; !ok || app == "" || !known {
			continue
		}
		out[app] = level
	}
	return out
}

func belowMinLevel(mins map[string]string, app, level string) bool {
	min, ok := mins[app]
	if !ok {
		for key, value := range mins {
			if strings.HasPrefix(app, key+"-") {
				min, ok = value, true
				break
			}
		}
	}
	return ok && levelRank[level] < levelRank[min]
}

func streamName(stream byte) string {
	if stream == streamStderr {
		return "stderr"
	}
	return "stdout"
}

func mapLine(app, containerID string, stream byte, ts time.Time, message string) entry {
	message = strings.TrimRight(message, "\r\n")
	if len(message) > maxLineBytes {
		message = message[:maxLineBytes]
	}
	e := entry{
		App:     app,
		Level:   detectLevel(message, stream),
		Message: message,
		Meta: map[string]string{
			"source":       "collector",
			"container_id": shortID(containerID),
			"stream":       streamName(stream),
		},
	}
	if !ts.IsZero() {
		e.TS = ts.Format(time.RFC3339Nano)
	}
	return e
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
