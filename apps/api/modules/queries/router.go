package queries

import "github.com/go-chi/chi/v5"

func RegisterRoutes(router chi.Router, service *Service) {
	handler := newHandler(service)
	router.Get("/queries", handler.list)
	router.Post("/queries", handler.create)
	router.Delete("/queries/{id}", handler.remove)
}
