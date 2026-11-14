package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/internal/authcrypto"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// IngestKeyVerifier authenticates a bearer token and returns the key, whose
// app and workspace scope the request.
type IngestKeyVerifier interface {
	VerifyIngestKey(ctx context.Context, token string) (schemas.APIKey, error)
}

// RequireIngestAuth guards the server and source-map ingest endpoints: it
// accepts the legacy shared token when one is set, which writes to the
// instance workspace only, then falls back to a per-app key, stamping the
// key's app and workspace into the request's ingest scope. The instance
// workspace's entries may be routed by app name; no other workspace's are.
func RequireIngestAuth(legacyToken string, keys IngestKeyVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			token, ok := authcrypto.BearerToken(request.Header.Get("Authorization"))
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("missing bearer token"))
				return
			}
			if legacyToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(legacyToken)) == 1 {
				legacy := authcontext.IngestScope{WorkspaceID: schemas.InstanceWorkspaceID, Routable: true}
				next.ServeHTTP(w, request.WithContext(authcontext.WithIngestScope(request.Context(), legacy)))
				return
			}
			key, err := keys.VerifyIngestKey(request.Context(), token)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			scoped := authcontext.WithIngestScope(request.Context(), authcontext.IngestScope{
				App: key.App, WorkspaceID: key.WorkspaceID, Routable: key.WorkspaceID == schemas.InstanceWorkspaceID,
			})
			next.ServeHTTP(w, request.WithContext(scoped))
		})
	}
}

// KeyByBearerTokenHash names the rate bucket by the hashed bearer token, so
// each key is limited independently.
func KeyByBearerTokenHash(request *http.Request) string {
	token, _ := authcrypto.BearerToken(request.Header.Get("Authorization"))
	return authcrypto.HashToken(token)
}
