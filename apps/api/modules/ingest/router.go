package ingest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, limiter, ingestAuth func(http.Handler) http.Handler) {
	handler := newHandler(service)
	router.With(limiter, ingestAuth).Post("/ingest", handler.ingest)
}
