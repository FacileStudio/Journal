package authcontext

import "context"

// Identity is the authenticated caller: who they are and whether they can
// reach the admin endpoints.
type Identity struct {
	UserID  int64
	Email   string
	IsAdmin bool
}

type contextKey struct{}

// With stores an Identity in the request context.
func With(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, identity)
}

// From reads the Identity stored by With, reporting false when the request
// carries none.
func From(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(contextKey{}).(Identity)
	return identity, ok
}

// IngestScope is the per-request ingest context. KeyID, DailyQuota and Origin
// are set by the browser endpoint only; a secret key leaves them zero and the
// quota is not consulted.
type IngestScope struct {
	App string

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
