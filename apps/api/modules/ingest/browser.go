package ingest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

const (
	maxBrowserEvents      = 20
	maxBrowserBodyBytes   = 128 << 10
	maxBrowserMetaBytes   = 8 << 10
	maxBrowserStringBytes = 2000
	maxBrowserStackBytes  = 8000
	maxBrowserArrayLen    = 50
	maxBrowserMetaDepth   = 6
	maxBrowserCount       = 10_000
)

// scrubbedKeys never reach the database, whatever an app puts in meta.
//
// A browser SDK collects whatever the page hands it, and pages hand over
// request payloads and form state without meaning to. Dropping these at the
// door is cheaper than explaining later why a session token is sitting in a
// log line that six people can read.
var scrubbedKeys = map[string]bool{
	"access_token":  true,
	"accesstoken":   true,
	"api_key":       true,
	"apikey":        true,
	"authorization": true,
	"cookie":        true,
	"credentials":   true,
	"jwt":           true,
	"passwd":        true,
	"password":      true,
	"refresh_token": true,
	"refreshtoken":  true,
	"secret":        true,
	"session":       true,
	"token":         true,
}

const scrubbedValue = "[scrubbed]"

// browser accepts error reports from a page.
//
// It is a separate route from /ingest rather than a flag on it because the two
// trust the caller to a completely different degree. Everything here is
// hostile until proven otherwise: the batch is small, the body is small, meta
// is scrubbed and capped, and the fields that matter for triage — app, origin,
// user agent — are stamped by the server, not read from the payload.
func (h *Handler) browser(w http.ResponseWriter, r *http.Request) {
	var req BrowserRequest
	if err := httpjson.DecodeJSONLimit(w, r, &req, maxBrowserBodyBytes); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if len(req.Events) == 0 {
		httpjson.WriteJSON(w, http.StatusCreated, IngestResponse{Ingested: 0})
		return
	}
	if len(req.Events) > maxBrowserEvents {
		httpjson.WriteError(w, errors.Invalid(fmt.Sprintf("batch exceeds the maximum of %d events", maxBrowserEvents)))
		return
	}

	scope, ok := authcontext.IngestScopeFrom(r.Context())
	if !ok || scope.App == "" {
		httpjson.WriteError(w, errors.Unauthorized("missing ingest scope"))
		return
	}

	now := time.Now().UTC()
	entries := make([]schemas.LogEntry, 0, len(req.Events))
	for _, event := range req.Events {
		if strings.TrimSpace(event.Message) == "" {
			httpjson.WriteError(w, errors.Invalid("message is required"))
			return
		}
		level := event.Level
		if level == "" {
			level = "error"
		}
		if !validLevels[level] {
			httpjson.WriteError(w, errors.Invalid("level must be one of debug, info, warn, error"))
			return
		}

		createdAt := now
		if event.Ts != "" {
			parsed, err := time.Parse(time.RFC3339, event.Ts)
			if err != nil {
				httpjson.WriteError(w, errors.Invalid("ts must be an RFC3339 timestamp"))
				return
			}
			createdAt = clampTimestamp(parsed.UTC(), now)
		}

		entries = append(entries, schemas.LogEntry{
			App:       scope.App,
			Level:     level,
			Message:   capMessage(event.Message),
			Meta:      browserMeta(event, req, scope, r),
			CreatedAt: createdAt,
		})
	}

	if err := h.service.ConsumeQuota(r.Context(), scope.KeyID, scope.DailyQuota, len(entries)); err != nil {
		if errors.Status(err) == http.StatusTooManyRequests {
			w.Header().Set("Retry-After", strconv.Itoa(secondsUntilUTCMidnight(now)))
		}
		httpjson.WriteError(w, err)
		return
	}

	ingested, err := h.service.Ingest(r.Context(), entries)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, IngestResponse{Ingested: ingested})
}

// browserMeta assembles the context the dashboard triages on.
//
// The client's own meta goes in first and the server's fields go in last, so a
// page cannot claim an origin, an app or a user agent it does not have. Every
// key here is flat and stable on purpose: the explorer pivots on meta keys.
//
// The batch owns the session id, so any session_id an event carries is deleted
// before the batch's is written: put skips an empty value, and without the
// delete a page could file its errors under another tab's session.
//
// An oversized meta falls back to the few keys triage cannot work without.
// session_id is one of them, for the same reason url is — a payload too big to
// store whole is exactly the noisy page whose other events are worth finding.
// It is written only when it exists, because the session index is partial on
// key presence and a null would pull every oversized event into it.
func browserMeta(event BrowserEvent, req BrowserRequest, scope authcontext.IngestScope, r *http.Request) map[string]any {
	meta := map[string]any{}
	for key, value := range scrubValue(event.Meta, 0).(map[string]any) {
		meta[key] = value
	}

	put := func(key, value string, limit int) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			meta[key] = capString(trimmed, limit)
		}
	}

	put("stack", event.Stack, maxBrowserStackBytes)
	put("url", event.URL, maxBrowserStringBytes)
	put("route", event.Route, maxBrowserStringBytes)
	put("kind", event.Kind, 64)
	put("release", req.Release, 200)
	put("environment", req.Environment, 64)
	delete(meta, "session_id")
	put("session_id", req.SessionID, 64)
	put("user_email", event.User.Email, 320)
	put("user_id", event.User.ID, 200)
	if event.Count > 1 {
		meta["count"] = min(event.Count, maxBrowserCount)
	}

	meta["source"] = "browser"
	meta["origin"] = scope.Origin
	put("user_agent", r.Header.Get("User-Agent"), 512)

	if encoded, err := json.Marshal(meta); err == nil && len(encoded) > maxBrowserMetaBytes {
		reduced := map[string]any{
			"source":     "browser",
			"origin":     scope.Origin,
			"url":        meta["url"],
			"stack":      capString(fmt.Sprint(meta["stack"]), maxBrowserStackBytes/2),
			"meta_error": "context exceeded 8KB and was dropped",
		}
		if session, ok := meta["session_id"]; ok {
			reduced["session_id"] = session
		}
		return reduced
	}
	return meta
}

// scrubValue walks the client's meta, redacting anything that smells like a
// credential and cutting the shape down to something a log store should hold.
func scrubValue(value any, depth int) any {
	if depth > maxBrowserMetaDepth {
		return "[too deep]"
	}
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			if scrubbedKeys[strings.ToLower(key)] {
				out[key] = scrubbedValue
				continue
			}
			out[key] = scrubValue(nested, depth+1)
		}
		return out
	case []any:
		if len(typed) > maxBrowserArrayLen {
			typed = typed[:maxBrowserArrayLen]
		}
		out := make([]any, 0, len(typed))
		for _, nested := range typed {
			out = append(out, scrubValue(nested, depth+1))
		}
		return out
	case string:
		return capString(typed, maxBrowserStringBytes)
	default:
		return value
	}
}

func capString(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	cut := limit
	for cut > 0 && !utf8Start(value[cut]) {
		cut--
	}
	return value[:cut] + "…"
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }

func secondsUntilUTCMidnight(now time.Time) int {
	midnight := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
	seconds := int(midnight.Sub(now).Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}
