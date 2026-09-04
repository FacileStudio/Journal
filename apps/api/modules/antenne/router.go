package antenne

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the Antenne settings endpoints under /api/settings/antenne.
func RegisterRoutes(r chi.Router, service *Service, logger *slog.Logger) {
	h := &handler{service: service, logger: logger}
	r.Route("/api/settings/antenne", func(r chi.Router) {
		r.Get("/", h.getSettings)
		r.Put("/", h.updateSettings)
	})
}