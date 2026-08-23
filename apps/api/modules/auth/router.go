package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the routes porte does not own.
//
// /auth/config, /auth/logout and the whole OIDC flow come from the kit; what
// is left here is the local password path and /auth/me, which reads a table
// porte knows nothing about.
//
// Under SSO_ONLY the credential routes are not registered rather than
// rejected, so there is no endpoint left to probe for an account. The delete
// route runs in both modes: erasure is a right of the account, not of the
// credential that created it.
func RegisterRoutes(router chi.Router, service *Service, ssoOnly bool, credentialLimiter, sessionLimiter, requireAuth func(http.Handler) http.Handler, clearCookie func(http.ResponseWriter, *http.Request)) {
	handler := newHandler(service, clearCookie)
	if !ssoOnly {
		router.With(credentialLimiter).Post("/auth/register", handler.register)
		router.With(credentialLimiter).Post("/auth/login", handler.login)
	}
	router.With(sessionLimiter, requireAuth).Get("/auth/me", handler.me)
	router.With(sessionLimiter, requireAuth).Delete("/auth/me", handler.deleteMe)
}
