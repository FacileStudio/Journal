package ingest

import (
	"github.com/FacileStudio/Journal/apps/api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, token string) {
	handler := newHandler(service)
	router.With(middleware.RequireToken(token)).Post("/ingest", handler.ingest)
}
