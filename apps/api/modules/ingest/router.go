package ingest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, limiter, ingestAuth func(http.Handler) http.Handler) {
	handler := newHandler(service)
	router.With(limiter, ingestAuth).Post("/ingest", handler.ingest)
}

// RegisterBrowserRoutes mounts the public endpoint separately, with its own
// limiter and its own auth, so no change to the server route can widen it by
// accident.
func RegisterBrowserRoutes(router chi.Router, service *Service, limiter, browserAuth func(http.Handler) http.Handler) {
	handler := newHandler(service)
	router.With(limiter, browserAuth).Post("/ingest/browser", handler.browser)
}
