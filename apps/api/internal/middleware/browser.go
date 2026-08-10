package middleware

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/internal/authcrypto"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/httprate"
)

// BrowserKeyVerifier resolves a public ingest key. It returns the whole record
// because authenticating the token is only half the check.
type BrowserKeyVerifier interface {
	VerifyBrowserKey(ctx context.Context, token string) (schemas.APIKey, error)
}

// RequireBrowserIngestAuth guards the one endpoint whose credential is public.
//
// Three things stand between a leaked key and a filled disk, and this
// middleware is two of them: the key must exist and be of the public kind, and
// the browser's Origin must be on that key's allowlist. The third is the daily
// quota, which the handler consumes before it writes.
//
// The origin check is a server-side authorization decision, not CORS. CORS
// only governs whether a script may *read* the response; the request itself is
// sent either way, so an allowlist enforced by response headers alone would
// stop nobody. The Access-Control-Allow-Origin below exists so the SDK can
// read its own 201, and it is set after the origin has already been approved.
func RequireBrowserIngestAuth(keys BrowserKeyVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			token := BrowserKeyToken(request)
			if token == "" {
				httpjson.WriteError(w, errors.Unauthorized("missing key"))
				return
			}

			key, err := keys.VerifyBrowserKey(request.Context(), token)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}

			origin := strings.TrimSpace(request.Header.Get("Origin"))
			if origin == "" {
				httpjson.WriteError(w, errors.Forbidden("this endpoint requires an Origin header"))
				return
			}
			if !OriginAllowed(origin, key.AllowedOrigins) {
				httpjson.WriteError(w, errors.Forbidden("origin "+origin+" is not allowed by this key"))
				return
			}

			header := w.Header()
			header.Add("Vary", "Origin")
			header.Set("Access-Control-Allow-Origin", origin)

			scoped := authcontext.WithIngestScope(request.Context(), authcontext.IngestScope{
				App:        key.App,
				KeyID:      key.ID,
				DailyQuota: key.DailyQuota,
				Origin:     origin,
			})
			next.ServeHTTP(w, request.WithContext(scoped))
		})
	}
}

// BrowserKeyToken reads the key from the query string, falling back to a
// bearer header.
//
// The query string is the primary transport because navigator.sendBeacon sets
// no headers, and a beacon on pagehide is the only way to keep the error from
// the navigation that caused it. The token is public by design, so putting it
// in a URL costs nothing that its presence in the bundle has not already cost.
func BrowserKeyToken(request *http.Request) string {
	if key := strings.TrimSpace(request.URL.Query().Get("key")); key != "" {
		return key
	}
	token, _ := authcrypto.BearerToken(request.Header.Get("Authorization"))
	return token
}

// OriginAllowed compares exactly. The stored list is already normalized to
// what a browser sends, so anything clever here would only widen it.
func OriginAllowed(origin string, allowed []string) bool {
	if len(allowed) == 0 {
		return false
	}
	return slices.Contains(allowed, strings.ToLower(origin))
}

// KeyByBrowserKeyAndIP buckets the browser rate limit per key *and* per IP.
//
// Per key alone would let one hostile page exhaust the limit for every real
// visitor; per IP alone would let a hundred keys share one bucket behind a
// corporate NAT. The pair is what a burst from one browser actually looks like.
//
// It shapes honest traffic and nothing more. The IP comes from chi's RealIP,
// which rewrites RemoteAddr from X-Forwarded-For without checking who the peer
// is, so anything that can set a header can mint itself a fresh bucket — see
// KeyByBrowserKey for the bound that holds anyway.
func KeyByBrowserKeyAndIP(request *http.Request) (string, error) {
	ip, err := httprate.KeyByIP(request)
	if err != nil {
		return "", err
	}
	return authcrypto.HashToken(BrowserKeyToken(request)) + "|" + ip, nil
}

// KeyByBrowserKey buckets on the credential alone.
//
// This is the ceiling that survives a spoofed X-Forwarded-For: the key is
// verified against the database, so no header rotates it. It is deliberately
// far above what any real page produces — it exists to bound write pressure
// when a public key leaks, not to shape traffic. The daily quota is the other
// bound that owes nothing to the network layer.
func KeyByBrowserKey(request *http.Request) (string, error) {
	return authcrypto.HashToken(BrowserKeyToken(request)), nil
}
