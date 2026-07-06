package auth

import (
	"net/http"

	"github.com/FacileStudio/Journal/apps/api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, allowRegistration bool, credentialLimiter, sessionLimiter func(http.Handler) http.Handler) {
	handler := newHandler(service, allowRegistration)
	router.With(sessionLimiter).Get("/auth/config", handler.config)
	router.With(credentialLimiter).Post("/auth/register", handler.register)
	router.With(credentialLimiter).Post("/auth/login", handler.login)
	router.With(sessionLimiter).Post("/auth/logout", handler.logout)
	router.With(sessionLimiter, middleware.RequireAuth(service)).Get("/auth/me", handler.me)
}
