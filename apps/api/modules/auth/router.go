package auth

import (
	"github.com/FacileStudio/Journal/apps/api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, allowRegistration bool) {
	handler := newHandler(service, allowRegistration)
	router.Get("/auth/config", handler.config)
	router.Post("/auth/register", handler.register)
	router.Post("/auth/login", handler.login)
	router.Post("/auth/logout", handler.logout)
	router.With(middleware.RequireAuth(service)).Get("/auth/me", handler.me)
}
