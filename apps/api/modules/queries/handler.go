package queries

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

	queries := make([]QueryResponse, 0, len(records))
	for _, record := range records {
		queries = append(queries, mapQuery(record))
	}
	httpjson.WriteJSON(w, http.StatusOK, ListResponse{Queries: queries})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	record, err := h.service.Create(r.Context(), req.Name, req.Params)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, CreateResponse{Query: mapQuery(*record)})
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

func mapQuery(record schemas.SavedQuery) QueryResponse {
	return QueryResponse{
		ID:        record.ID,
		Name:      record.Name,
		Params:    record.Params,
		CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339),
	}
}
