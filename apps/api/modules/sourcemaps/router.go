package sourcemaps

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterUploadRoutes mounts the half an application writes to, behind the
// same per-app credential it already ships logs with.
func RegisterUploadRoutes(router chi.Router, service *Service, limiter, ingestAuth func(http.Handler) http.Handler) {
	handler := newHandler(service)
	router.With(limiter, ingestAuth).Post("/sourcemaps", handler.upload)
	router.With(limiter, ingestAuth).Get("/sourcemaps", handler.list)
}

// RegisterAdminRoutes mounts the half a human reads, behind a session.
func RegisterAdminRoutes(router chi.Router, service *Service) {
	handler := newHandler(service)
	router.Get("/sourcemaps/releases", handler.releases)
	router.Delete("/sourcemaps", handler.remove)
}
