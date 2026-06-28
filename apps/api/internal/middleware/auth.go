package middleware

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/internal/httpjson"
)

type Authenticator interface {
	Authenticate(ctx context.Context, authorization string) (authcontext.Identity, error)
}

func RequireAuth(authenticator Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			identity, err := authenticator.Authenticate(request.Context(), request.Header.Get("Authorization"))
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			next.ServeHTTP(w, request.WithContext(authcontext.With(request.Context(), identity)))
		})
	}
}
