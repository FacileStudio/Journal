package deletions

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts Registre's signed deletion push. It sits outside the
// session and CSRF middleware on purpose: the caller is Registre, and the
// handler authenticates the notice by its signature.
func RegisterRoutes(router chi.Router, service *Service) {
	router.Post("/registre/deletions", service.Handler().ServeHTTP)
}
