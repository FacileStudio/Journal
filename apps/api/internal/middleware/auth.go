package middleware

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// IdentityResolver turns the user id porte authenticated into Journal's own
// identity.
type IdentityResolver interface {
	IdentityForUser(ctx context.Context, userID int64) (authcontext.Identity, error)
}

// RequireAuth runs behind porte's own middleware and hydrates what porte
// deliberately does not carry.
//
// porte authenticates the credential — cookie or bearer, one hashed session
// row, one expiry, one idle window — and hands on a user id. It holds no email
// and no is_admin, because a library that decided what an admin may do would
// be routed around by the second app that adopted it. So the profile is looked
// up here, from the table this app owns, and lands in the context every
// handler and RequireAdmin already read. Nothing downstream changed.
func RequireAuth(kit portekit, resolver IdentityResolver) func(http.Handler) http.Handler {
	hydrate := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			authenticated, ok := porte.From(request.Context())
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("missing auth token"))
				return
			}
			identity, err := resolver.IdentityForUser(request.Context(), authenticated.UserID)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			next.ServeHTTP(w, request.WithContext(authcontext.With(request.Context(), identity)))
		})
	}
	return func(next http.Handler) http.Handler {
		return kit.RequireAuth(hydrate(next))
	}
}

// portekit is the one method this package needs from *porte/oidc.Kit, named as
// an interface so the middleware stays testable without a provider.
type portekit interface {
	RequireAuth(http.Handler) http.Handler
}
