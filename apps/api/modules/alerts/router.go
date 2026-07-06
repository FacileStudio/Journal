package alerts

import "github.com/go-chi/chi/v5"

func RegisterRoutes(router chi.Router, service *Service) {
	handler := newHandler(service)
	router.Get("/alerts", handler.list)
	router.Post("/alerts", handler.create)
	router.Patch("/alerts/{id}", handler.update)
	router.Delete("/alerts/{id}", handler.remove)
}
