package ingest

import (
	"net/http"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/internal/httpjson"
	"github.com/FacileStudio/Journal/apps/api/schemas"
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
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	raw := req.Entries
	if len(raw) == 0 {
		raw = []IngestEntry{{App: req.App, Level: req.Level, Message: req.Message, Ts: req.Ts, Meta: req.Meta}}
	}

	now := time.Now().UTC()
	entries := make([]schemas.LogEntry, 0, len(raw))
	for _, entry := range raw {
		if entry.App == "" {
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
			createdAt = parsed.UTC()
		}
		entries = append(entries, schemas.LogEntry{
			App:       entry.App,
			Level:     level,
			Message:   entry.Message,
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
