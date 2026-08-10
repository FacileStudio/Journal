package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

type stubKeys struct {
	key schemas.APIKey
	err error
}

func (s stubKeys) VerifyBrowserKey(context.Context, string) (schemas.APIKey, error) {
	return s.key, s.err
}

func publicKey() schemas.APIKey {
	return schemas.APIKey{
		ID:             7,
		App:            "shop",
		Kind:           schemas.KeyKindPublic,
		AllowedOrigins: []string{"https://shop.example"},
		DailyQuota:     5000,
	}
}

func call(t *testing.T, keys BrowserKeyVerifier, target, origin string) (*httptest.ResponseRecorder, *authcontext.IngestScope) {
	t.Helper()
	var seen *authcontext.IngestScope
	handler := RequireBrowserIngestAuth(keys)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if scope, ok := authcontext.IngestScopeFrom(r.Context()); ok {
			seen = &scope
		}
		w.WriteHeader(http.StatusCreated)
	}))

	request := httptest.NewRequest(http.MethodPost, target, nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder, seen
}

func TestBrowserAuthAcceptsAnAllowedOrigin(t *testing.T) {
	recorder, scope := call(t, stubKeys{key: publicKey()}, "/ingest/browser?key=journal_pub_shop_x", "https://shop.example")

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if scope == nil || scope.App != "shop" || scope.KeyID != 7 || scope.DailyQuota != 5000 {
		t.Fatalf("scope = %+v, want the key's app, id and quota", scope)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://shop.example" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want the request's origin", got)
	}
}

// The origin check is the authorization decision. A request from an origin the
// key does not list must be refused before it reaches the handler — not merely
// left unreadable by the browser, which is all CORS would do.
func TestBrowserAuthRefusesAnUnlistedOrigin(t *testing.T) {
	recorder, scope := call(t, stubKeys{key: publicKey()}, "/ingest/browser?key=journal_pub_shop_x", "https://evil.example")

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	if scope != nil {
		t.Fatal("the handler ran for an unlisted origin")
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("a refused origin was echoed back as allowed")
	}
}

func TestBrowserAuthRefusesWithoutOrigin(t *testing.T) {
	recorder, _ := call(t, stubKeys{key: publicKey()}, "/ingest/browser?key=journal_pub_shop_x", "")
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestBrowserAuthRefusesWithoutKey(t *testing.T) {
	recorder, _ := call(t, stubKeys{key: publicKey()}, "/ingest/browser", "https://shop.example")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestBrowserAuthPropagatesAnInvalidKey(t *testing.T) {
	keys := stubKeys{err: errors.Unauthorized("invalid ingest token")}
	recorder, _ := call(t, keys, "/ingest/browser?key=nope", "https://shop.example")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

// An empty allowlist means "nothing", never "everything": a key whose origins
// were never set must be inert, not open.
func TestOriginAllowed(t *testing.T) {
	allowed := []string{"https://shop.example", "http://localhost:5173"}
	cases := []struct {
		origin string
		want   bool
	}{
		{"https://shop.example", true},
		{"http://localhost:5173", true},
		{"HTTPS://SHOP.EXAMPLE", true},
		{"https://shop.example:443", false},
		{"https://sub.shop.example", false},
		{"http://shop.example", false},
		{"null", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := OriginAllowed(tc.origin, allowed); got != tc.want {
			t.Fatalf("OriginAllowed(%q) = %v, want %v", tc.origin, got, tc.want)
		}
	}
	if OriginAllowed("https://shop.example", nil) {
		t.Fatal("an empty allowlist accepted an origin")
	}
}

// Two pages on two keys must not share a rate-limit bucket, and neither must
// two visitors on one key.
func TestKeyByBrowserKeyAndIP(t *testing.T) {
	one := httptest.NewRequest(http.MethodPost, "/ingest/browser?key=a", nil)
	one.RemoteAddr = "10.0.0.1:1234"
	two := httptest.NewRequest(http.MethodPost, "/ingest/browser?key=b", nil)
	two.RemoteAddr = "10.0.0.1:1234"
	three := httptest.NewRequest(http.MethodPost, "/ingest/browser?key=a", nil)
	three.RemoteAddr = "10.0.0.2:1234"

	first, err := KeyByBrowserKeyAndIP(one)
	if err != nil {
		t.Fatalf("KeyByBrowserKeyAndIP: %v", err)
	}
	second, _ := KeyByBrowserKeyAndIP(two)
	third, _ := KeyByBrowserKeyAndIP(three)

	if first == second {
		t.Fatal("two keys from one IP share a bucket")
	}
	if first == third {
		t.Fatal("two IPs on one key share a bucket")
	}
}
