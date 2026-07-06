package apikeys

import "github.com/go-chi/chi/v5"

func RegisterRoutes(router chi.Router, service *Service) {
	handler := newHandler(service)
	router.Get("/apikeys", handler.list)
	router.Post("/apikeys", handler.create)
	router.Delete("/apikeys/{id}", handler.revoke)
}
