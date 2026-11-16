package deletions

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts Registre's signed deletion push. It sits outside the
// session and CSRF middleware on purpose: the caller is Registre, and the
// handler authenticates the notice by its signature.
func RegisterRoutes(router chi.Router, handler http.Handler) {
	router.Post("/registre/deletions", handler.ServeHTTP)
}
