package logs

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/internal/httpjson"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"github.com/go-chi/chi/v5"
)

const (
	defaultContextSize  = 50
	maxContextSize      = 200
	maxHistogramBuckets = 90
)

var histogramBucketOptions = []int64{60, 300, 900, 3600, 21600, 86400}

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

	var nextBefore *Cursor
	if len(records) == params.Limit && len(records) > 0 {
		last := records[len(records)-1]
		nextBefore = &Cursor{Ts: last.CreatedAt.UTC().Format(time.RFC3339Nano), ID: last.ID}
	}

	httpjson.WriteJSON(w, http.StatusOK, ListResponse{Entries: entries, NextBefore: nextBefore})
}

func (h *Handler) histogram(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	if params.Until == nil {
		now := time.Now().UTC()
		params.Until = &now
	}
	if params.Since == nil {
		since := params.Until.Add(-24 * time.Hour)
		params.Since = &since
	}
	if !params.Until.After(*params.Since) {
		httpjson.WriteError(w, errors.Invalid("until must be after since"))
		return
	}

	bucketSeconds := pickBucketSeconds(int64(params.Until.Sub(*params.Since) / time.Second))
	buckets, err := h.service.Histogram(r.Context(), params, bucketSeconds)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, HistogramResponse{BucketSeconds: bucketSeconds, Buckets: buckets})
}

func (h *Handler) logContext(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("id must be an integer"))
		return
	}

	q := r.URL.Query()
	before := parseContextSize(q.Get("before"))
	after := parseContextSize(q.Get("after"))

	records, err := h.service.Context(r.Context(), id, before, after)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	entries := make([]LogResponse, 0, len(records))
	for _, record := range records {
		entries = append(entries, mapEntry(record))
	}

	httpjson.WriteJSON(w, http.StatusOK, ContextResponse{Entries: entries, AnchorID: id})
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
		App:       q.Get("app"),
		Query:     q.Get("q"),
		RequestID: q.Get("request_id"),
		Limit:     100,
		Levels:    parseLevels(q["level"]),
	}

	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			params.Limit = n
		}
	}
	if params.Limit > 1000 {
		params.Limit = 1000
	}

	beforeTs := q.Get("before_ts")
	beforeID := q.Get("before_id")
	if (beforeTs == "") != (beforeID == "") {
		return ListParams{}, errors.Invalid("before_ts and before_id must be provided together")
	}
	if beforeTs != "" {
		parsedTs, err := time.Parse(time.RFC3339Nano, beforeTs)
		if err != nil {
			return ListParams{}, errors.Invalid("before_ts must be an RFC3339 timestamp")
		}
		parsedID, err := strconv.ParseInt(beforeID, 10, 64)
		if err != nil {
			return ListParams{}, errors.Invalid("before_id must be an integer")
		}
		ts := parsedTs.UTC()
		params.BeforeTs = &ts
		params.BeforeID = &parsedID
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

func parseContextSize(raw string) int {
	size := defaultContextSize
	if raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			size = n
		}
	}
	if size > maxContextSize {
		size = maxContextSize
	}
	return size
}

func pickBucketSeconds(rangeSeconds int64) int64 {
	for _, bucket := range histogramBucketOptions {
		if rangeSeconds <= bucket*maxHistogramBuckets {
			return bucket
		}
	}
	return histogramBucketOptions[len(histogramBucketOptions)-1]
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
