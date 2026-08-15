package logs

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts the log endpoints under /logs and /apps.
func RegisterRoutes(router chi.Router, service *Service, stacks StackResolver) {
	handler := newHandler(service, stacks)
	router.Get("/logs", handler.list)
	router.Get("/logs/histogram", handler.histogram)
	router.Get("/logs/{id}/context", handler.logContext)
	router.Get("/logs/{id}/stack", handler.stack)
	router.Get("/apps", handler.apps)
}
