package logs

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/internal/httpjson"
	"github.com/FacileStudio/Journal/apps/api/schemas"
)

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	records, err := h.service.List(r.Context(), params)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	entries := make([]LogResponse, 0, len(records))
	for _, record := range records {
		entries = append(entries, mapEntry(record))
	}

	var nextBefore *int64
	if len(records) == params.Limit && len(records) > 0 {
		last := records[len(records)-1].ID
		nextBefore = &last
	}

	httpjson.WriteJSON(w, http.StatusOK, ListResponse{Entries: entries, NextBefore: nextBefore})
}

func (h *Handler) apps(w http.ResponseWriter, r *http.Request) {
	apps, err := h.service.Apps(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, AppsResponse{Apps: apps})
}

func parseListParams(r *http.Request) (ListParams, error) {
	q := r.URL.Query()
	params := ListParams{
		App:    q.Get("app"),
		Query:  q.Get("q"),
		Limit:  100,
		Levels: parseLevels(q["level"]),
	}

	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			params.Limit = n
		}
	}
	if params.Limit > 1000 {
		params.Limit = 1000
	}

	if raw := q.Get("before"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return ListParams{}, errors.Invalid("before must be an integer cursor")
		}
		params.Before = &n
	}

	if raw := q.Get("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return ListParams{}, errors.Invalid("since must be an RFC3339 timestamp")
		}
		t := parsed.UTC()
		params.Since = &t
	}
	if raw := q.Get("until"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return ListParams{}, errors.Invalid("until must be an RFC3339 timestamp")
		}
		t := parsed.UTC()
		params.Until = &t
	}

	return params, nil
}

func parseLevels(values []string) []string {
	levels := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				levels = append(levels, part)
			}
		}
	}
	return levels
}

func mapEntry(record schemas.LogEntry) LogResponse {
	return LogResponse{
		ID:         record.ID,
		App:        record.App,
		Level:      record.Level,
		Message:    record.Message,
		Meta:       record.Meta,
		CreatedAt:  record.CreatedAt.UTC().Format(time.RFC3339),
		ReceivedAt: record.ReceivedAt.UTC().Format(time.RFC3339),
	}
}
