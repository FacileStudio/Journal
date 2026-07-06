package main

import (
	"encoding/json"
	"strings"
	"time"
)

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
	case "debug", "trace":
		return "debug"
	case "info":
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
	if stream == streamStderr {
		return "error"
	}
	return "info"
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
