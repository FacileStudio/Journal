package ingest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the server and browser ingest endpoints.
func RegisterRoutes(router chi.Router, service *Service, limiter, ingestAuth func(http.Handler) http.Handler) {
	handler := newHandler(service)
	router.With(limiter, ingestAuth).Post("/ingest", handler.ingest)
}

// RegisterBrowserRoutes mounts the public endpoint separately, with its own
// limiters and its own auth, so no change to the server route can widen it by
// accident.
func RegisterBrowserRoutes(router chi.Router, service *Service, browserAuth func(http.Handler) http.Handler, limiters ...func(http.Handler) http.Handler) {
	handler := newHandler(service)
	router.With(append(limiters, browserAuth)...).Post("/ingest/browser", handler.browser)
}
