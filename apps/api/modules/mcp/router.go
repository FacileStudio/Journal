package mcp

import (
	mcpkit "github.com/FacileStudio/tronc/mcp"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the MCP endpoint at the resource's path and its RFC
// 9728 metadata on router, the root router outside /api.
func RegisterRoutes(router chi.Router, service *Service) {
	service.server.Mount(router)
}

// RegisterPendingRoutes mounts the confirm page's API under /mcp/pending on
// router, which the caller has already put behind the session.
func RegisterPendingRoutes(router chi.Router, service *Service) {
	router.Route("/mcp/pending", service.pending().Routes)
}

// pending is the confirm page, or nil on a nil service.
func (s *Service) pending() *mcpkit.PendingPage[PendingItem] {
	if s == nil {
		return nil
	}
	return s.page
}
