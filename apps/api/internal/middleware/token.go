package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/internal/authcrypto"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// IngestKeyVerifier authenticates a bearer token and returns the app it
// belongs to.
type IngestKeyVerifier interface {
	VerifyIngestKey(ctx context.Context, token string) (string, error)
}

// RequireIngestAuth guards the server and source-map ingest endpoints: it
// accepts the legacy shared token when one is set, then falls back to a
// per-app key, stamping the app into the request's ingest scope.
func RequireIngestAuth(legacyToken string, keys IngestKeyVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			token, ok := authcrypto.BearerToken(request.Header.Get("Authorization"))
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("missing bearer token"))
				return
			}
			if legacyToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(legacyToken)) == 1 {
				next.ServeHTTP(w, request)
				return
			}
			app, err := keys.VerifyIngestKey(request.Context(), token)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			scoped := authcontext.WithIngestScope(request.Context(), authcontext.IngestScope{App: app})
			next.ServeHTTP(w, request.WithContext(scoped))
		})
	}
}

// KeyByBearerTokenHash names the rate bucket by the hashed bearer token, so
// each key is limited independently.
func KeyByBearerTokenHash(request *http.Request) (string, error) {
	token, _ := authcrypto.BearerToken(request.Header.Get("Authorization"))
	return authcrypto.HashToken(token), nil
}
