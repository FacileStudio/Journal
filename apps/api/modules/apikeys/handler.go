package apikeys

import (
	"net/http"
	"strconv"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/internal/httpjson"
	"github.com/FacileStudio/Journal/apps/api/schemas"

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

	keys := make([]KeyResponse, 0, len(records))
	for _, record := range records {
		keys = append(keys, mapKey(record))
	}
	httpjson.WriteJSON(w, http.StatusOK, ListResponse{Keys: keys})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	key, token, err := h.service.Create(r.Context(), req.App)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, CreateResponse{Key: mapKey(*key), Token: token})
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("id must be an integer"))
		return
	}

	if err := h.service.Revoke(r.Context(), id); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusNoContent, nil)
}

func mapKey(key schemas.APIKey) KeyResponse {
	var revokedAt *string
	if key.RevokedAt != nil {
		formatted := key.RevokedAt.UTC().Format(time.RFC3339)
		revokedAt = &formatted
	}
	return KeyResponse{
		ID:        key.ID,
		App:       key.App,
		Prefix:    key.Prefix,
		CreatedAt: key.CreatedAt.UTC().Format(time.RFC3339),
		RevokedAt: revokedAt,
	}
}
