// Package workspaces serves the workspaces a signed-in account belongs to.
package workspaces

import "github.com/go-chi/chi/v5"

// RegisterRoutes mounts GET /workspaces on a router that already runs the
// session check and workspace.Middleware.
func RegisterRoutes(router chi.Router) {
	router.Get("/workspaces", list)
}
