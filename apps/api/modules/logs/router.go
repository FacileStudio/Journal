package logs

import "github.com/go-chi/chi/v5"

func RegisterRoutes(router chi.Router, service *Service) {
	handler := newHandler(service)
	router.Get("/logs", handler.list)
	router.Get("/logs/histogram", handler.histogram)
	router.Get("/logs/{id}/context", handler.logContext)
	router.Get("/apps", handler.apps)
}
