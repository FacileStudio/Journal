package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the routes porte does not own.
//
// /auth/config, /auth/logout and the whole OIDC flow come from the kit; what
// is left here is /auth/me, which reads a table porte knows nothing about.
// Sign-in goes through Registre only, so no credential route exists to probe.
func RegisterRoutes(router chi.Router, service *Service, sessionLimiter, requireAuth func(http.Handler) http.Handler, clearCookie func(http.ResponseWriter, *http.Request)) {
	handler := newHandler(service, clearCookie)
	router.With(sessionLimiter, requireAuth).Get("/auth/me", handler.me)
	router.With(sessionLimiter, requireAuth).Delete("/auth/me", handler.deleteMe)
}
