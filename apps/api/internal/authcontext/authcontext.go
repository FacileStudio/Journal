package authcontext

import "context"

// IngestScope is the per-request ingest context. WorkspaceID is the
// workspace every entry of the request is written to: the key's own, or the
// instance workspace for the legacy token, which alone may be routed by app
// name (Routable). KeyID, DailyQuota and Origin are set by the browser
// endpoint only; a secret key leaves them zero and the key quota is not
// consulted.
type IngestScope struct {
	App         string
	WorkspaceID int64
	Routable    bool

	KeyID      int64
	DailyQuota int
	Origin     string
}

type ingestScopeKey struct{}

// WithIngestScope stores an IngestScope in the request context.
func WithIngestScope(ctx context.Context, scope IngestScope) context.Context {
	return context.WithValue(ctx, ingestScopeKey{}, scope)
}

// IngestScopeFrom reads the IngestScope stored by WithIngestScope, reporting
// false when the request carries none.
func IngestScopeFrom(ctx context.Context) (IngestScope, bool) {
	scope, ok := ctx.Value(ingestScopeKey{}).(IngestScope)
	return scope, ok
}
