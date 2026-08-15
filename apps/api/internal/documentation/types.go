// Package docs holds Journal's hand-written route registry, the single source
// the reference page at /docs and its OpenAPI document are both built from.
//
// The types are aliases of tronc/apiref's, so the registry is exactly the
// suite-wide shape and apiref.Undocumented can check it against the live router.
//
// The package is named docs rather than documentation because go/build ignores
// every file declaring `package documentation` — a godoc convention for
// documentation-only directories — which makes the package silently vanish with
// "build constraints exclude all Go files".
package docs

import "github.com/FacileStudio/tronc/apiref"

type (
	// Response names the registry that aggregates a module's declared routes.
	Response = apiref.Registry

	// Module is one module's route group.
	Module = apiref.Module

	// Route is one declared endpoint.
	Route = apiref.Route

	// Field is one parameter or body field.
	Field = apiref.Field

	// Error is one documented failure response.
	Error = apiref.Error
)
