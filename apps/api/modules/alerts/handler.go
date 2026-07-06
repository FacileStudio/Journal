package alerts

import (
	"net/http"
	"strconv"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/internal/httpjson"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	records, err := h.service.List(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	alerts := make([]AlertResponse, 0, len(records))
	for _, record := range records {
		alerts = append(alerts, mapAlert(record))
	}
	httpjson.WriteJSON(w, http.StatusOK, ListResponse{Alerts: alerts})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	record, err := h.service.Create(r.Context(), req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, AlertEnvelope{Alert: mapAlert(*record)})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("id must be an integer"))
		return
	}

	var req UpdateRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if req.Enabled == nil {
		httpjson.WriteError(w, errors.Invalid("enabled is required"))
		return
	}

	record, err := h.service.SetEnabled(r.Context(), id, *req.Enabled)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, AlertEnvelope{Alert: mapAlert(*record)})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("id must be an integer"))
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusNoContent, nil)
}

func mapAlert(record ruleRecord) AlertResponse {
	var lastFiredAt *string
	if record.LastFiredAt != nil {
		formatted := record.LastFiredAt.UTC().Format(time.RFC3339)
		lastFiredAt = &formatted
	}
	return AlertResponse{
		ID:            record.ID,
		Name:          record.Name,
		SavedQueryID:  record.SavedQueryID,
		QueryName:     record.QueryName,
		Threshold:     record.Threshold,
		WindowMinutes: record.WindowMinutes,
		WebhookURL:    record.WebhookURL,
		WebhookHeader: record.WebhookHeader,
		Enabled:       record.Enabled,
		LastFiredAt:   lastFiredAt,
		CreatedAt:     record.CreatedAt.UTC().Format(time.RFC3339),
	}
}
