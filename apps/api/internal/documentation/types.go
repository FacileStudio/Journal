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
	Response = apiref.Registry
	Module   = apiref.Module
	Route    = apiref.Route
	Field    = apiref.Field
	Error    = apiref.Error
)
