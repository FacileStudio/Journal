package ingest

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/internal/httpjson"
	"github.com/FacileStudio/Journal/apps/api/schemas"
)

const (
	maxMessageBytes      = 64 * 1024
	truncationSuffix     = " [truncated]"
	maxFutureTimestamp   = 5 * time.Minute
	maxBatchEntries      = 1000
	maxDecompressedBytes = 32 << 20
)

var validLevels = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) {
	var req IngestRequest
	var decodeErr error
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("Content-Encoding")), "gzip") {
		decodeErr = httpjson.DecodeGzipJSON(w, r, &req, maxDecompressedBytes)
	} else {
		decodeErr = httpjson.DecodeJSON(w, r, &req)
	}
	if decodeErr != nil {
		httpjson.WriteError(w, decodeErr)
		return
	}

	if len(req.Entries) > maxBatchEntries {
		httpjson.WriteError(w, errors.Invalid(fmt.Sprintf("batch exceeds the maximum of %d entries", maxBatchEntries)))
		return
	}

	raw := req.Entries
	if req.Entries == nil {
		raw = []IngestEntry{{App: req.App, Level: req.Level, Message: req.Message, Ts: req.Ts, Meta: req.Meta}}
	}

	scope, _ := authcontext.IngestScopeFrom(r.Context())
	now := time.Now().UTC()
	entries := make([]schemas.LogEntry, 0, len(raw))
	for _, entry := range raw {
		app := entry.App
		if scope.App != "" {
			if app == "" {
				app = scope.App
			} else if app != scope.App {
				httpjson.WriteError(w, errors.Invalid(fmt.Sprintf("app %q is not allowed by this API key (scoped to %q)", entry.App, scope.App)))
				return
			}
		}
		if app == "" {
			httpjson.WriteError(w, errors.Invalid("app is required"))
			return
		}
		if entry.Message == "" {
			httpjson.WriteError(w, errors.Invalid("message is required"))
			return
		}
		level := entry.Level
		if level == "" {
			level = "info"
		}
		if !validLevels[level] {
			httpjson.WriteError(w, errors.Invalid("level must be one of debug, info, warn, error"))
			return
		}
		createdAt := now
		if entry.Ts != "" {
			parsed, err := time.Parse(time.RFC3339, entry.Ts)
			if err != nil {
				httpjson.WriteError(w, errors.Invalid("ts must be an RFC3339 timestamp"))
				return
			}
			createdAt = clampTimestamp(parsed.UTC(), now)
		}
		entries = append(entries, schemas.LogEntry{
			App:       app,
			Level:     level,
			Message:   capMessage(entry.Message),
			Meta:      entry.Meta,
			CreatedAt: createdAt,
		})
	}

	ingested, err := h.service.Ingest(r.Context(), entries)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, IngestResponse{Ingested: ingested})
}

func clampTimestamp(parsed, now time.Time) time.Time {
	if parsed.After(now.Add(maxFutureTimestamp)) {
		return now
	}
	return parsed
}

func capMessage(message string) string {
	if len(message) <= maxMessageBytes {
		return message
	}
	cut := maxMessageBytes
	for cut > 0 && !utf8.RuneStart(message[cut]) {
		cut--
	}
	return message[:cut] + truncationSuffix
}
