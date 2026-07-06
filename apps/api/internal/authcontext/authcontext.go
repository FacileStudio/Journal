package authcontext

import "context"

type Identity struct {
	UserID  int64
	Email   string
	IsAdmin bool
}

type contextKey struct{}

func With(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, identity)
}

func From(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(contextKey{}).(Identity)
	return identity, ok
}

type IngestScope struct {
	App string
}

type ingestScopeKey struct{}

func WithIngestScope(ctx context.Context, scope IngestScope) context.Context {
	return context.WithValue(ctx, ingestScopeKey{}, scope)
}

func IngestScopeFrom(ctx context.Context) (IngestScope, bool) {
	scope, ok := ctx.Value(ingestScopeKey{}).(IngestScope)
	return scope, ok
}
