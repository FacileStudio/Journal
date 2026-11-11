package mcp

import (
	"encoding/json"
	stderrors "errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/logfilter"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

const (
	listMessageBytes = 1_000
	fullMessageBytes = 16_000
	listMetaBytes    = 2_000
	fullMetaBytes    = 32_000
	defaultWindow    = 24 * time.Hour
)

var (
	levels       = []string{"debug", "info", "warn", "error"}
	errLevel     = stderrors.New("levels must be among debug, info, warn and error")
	errSource    = stderrors.New(`source must be "server" or "browser"`)
	errWindow    = stderrors.New("until must be after since")
	errTimeValue = stderrors.New("since and until take an RFC 3339 time such as 2026-11-11T09:00:00Z or a duration such as 15m, 6h or 7d")
)

// shape is how much of an entry a tool shows.
type shape struct {
	messageBytes int
	metaBytes    int
	meta         bool
}

var (
	concise  = shape{messageBytes: listMessageBytes}
	detailed = shape{messageBytes: listMessageBytes, metaBytes: listMetaBytes, meta: true}
	full     = shape{messageBytes: fullMessageBytes, metaBytes: fullMetaBytes, meta: true}
)

// render shapes one entry: the message clipped, the request id and source
// lifted out of meta, and meta itself only when the shape asks for it, as a
// preview string once it outgrows the shape's budget.
func render(e schemas.LogEntry, s shape) entryOut {
	message, cut := mcpkit.Clip(e.Message, s.messageBytes)
	out := entryOut{
		ID: e.ID, App: e.App, Level: e.Level, Message: message, MessageTruncated: cut,
		CreatedAt: stamp(e.CreatedAt), RequestID: metaText(e.Meta, "request_id"), Source: metaText(e.Meta, "source"),
	}
	if !s.meta || len(e.Meta) == 0 {
		return out
	}
	raw, _ := json.Marshal(e.Meta)
	if len(raw) <= s.metaBytes {
		out.Meta = e.Meta
		return out
	}
	out.MetaPreview, out.MetaTruncated = mcpkit.Clip(string(raw), s.metaBytes)
	return out
}

// metaText is a string field of meta, empty when absent or not a string.
func metaText(meta map[string]any, key string) string {
	value, _ := meta[key].(string)
	return value
}

// stamp renders an instant in UTC.
func stamp(at time.Time) string {
	return at.UTC().Format(time.RFC3339)
}

// filterOf validates a tool's filter and turns it into the explorer's,
// filling the default window of the last 24 hours.
func filterOf(in filterInput, now time.Time) (logfilter.Params, error) {
	for _, level := range in.Levels {
		if !slices.Contains(levels, level) {
			return logfilter.Params{}, errLevel
		}
	}
	if in.Source != "" && in.Source != "server" && in.Source != "browser" {
		return logfilter.Params{}, errSource
	}
	until, err := instant(in.Until, now, now)
	if err != nil {
		return logfilter.Params{}, err
	}
	since, err := instant(in.Since, now, until.Add(-defaultWindow))
	if err != nil {
		return logfilter.Params{}, err
	}
	if !until.After(since) {
		return logfilter.Params{}, errWindow
	}
	return logfilter.Params{
		App: strings.TrimSpace(in.App), Levels: in.Levels, Query: strings.TrimSpace(in.Q), Source: in.Source,
		RequestID: strings.TrimSpace(in.RequestID), SessionID: strings.TrimSpace(in.SessionID), Since: &since, Until: &until,
	}, nil
}

// instant reads an RFC 3339 time or a duration back from now, fallback when
// empty.
func instant(raw string, now, fallback time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback.UTC(), nil
	}
	if at, err := time.Parse(time.RFC3339, raw); err == nil {
		return at.UTC(), nil
	}
	ago, err := durationOf(raw)
	if err != nil || ago <= 0 {
		return time.Time{}, errTimeValue
	}
	return now.Add(-ago).UTC(), nil
}

// durationOf reads a Go duration, plus whole days written as 7d.
func durationOf(raw string) (time.Duration, error) {
	if days, ok := strings.CutSuffix(raw, "d"); ok {
		n, err := strconv.Atoi(days)
		if err != nil {
			return 0, err
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(raw)
}
