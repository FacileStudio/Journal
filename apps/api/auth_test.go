package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/modules/apikeys"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/session"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

const testIngestToken = "legacy-ingest-token"

type liveApp struct {
	router   chi.Router
	db       *gorm.DB
	sessions *session.Manager
}

// newLiveApp is the real router over a real database, which is what it takes
// to say anything about the auth path: the credential is a row, the middleware
// is a query against it, and the identity the handlers read is a second query
// against a table porte does not know about. All three are database behaviour.
func newLiveApp(t *testing.T) liveApp {
	t.Helper()
	db := testdb.Migrated(t)
	appEnv := env.Config{IngestToken: testIngestToken}
	oidcKit, sessions, users := testKitFor(t, appEnv, sqlHandle(t, db))
	router := buildRouter(db, oidcKit, sessions, users, appEnv, slog.New(slog.DiscardHandler), nil, ssoRoutes{})
	return liveApp{router: router, db: db, sessions: sessions}
}

// signIn creates an account the way a first Facile sign-in leaves it and
// returns a session token for it, since sign-in itself is Registre's.
func (a liveApp) signIn(t *testing.T, email string, admin bool) (int64, string) {
	t.Helper()
	user := schemas.User{Email: email, Name: "Someone", IsAdmin: admin}
	if err := a.db.Create(&user).Error; err != nil {
		t.Fatalf("seed %s: %v", email, err)
	}
	token, _, err := a.sessions.Issue(context.Background(), user.ID, "test")
	if err != nil {
		t.Fatalf("issue a session: %v", err)
	}
	return user.ID, token
}

func call(t *testing.T, router chi.Router, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload := bytes.NewReader(nil)
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		payload = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, payload)
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// A session porte issued authenticates, Journal hydrates the profile porte does
// not carry, RequireAdmin reads is_admin out of it, and porte's logout revokes
// the session.
func TestSessionRunsThroughPorteEndToEnd(t *testing.T) {
	app := newLiveApp(t)
	_, admin := app.signIn(t, "first@facile.studio", true)
	_, member := app.signIn(t, "second@facile.studio", false)

	if code := call(t, app.router, http.MethodGet, "/api/auth/me", admin, nil).Code; code != http.StatusOK {
		t.Fatalf("me = %d", code)
	}
	if code := call(t, app.router, http.MethodGet, "/api/apikeys", admin, nil).Code; code != http.StatusOK {
		t.Fatalf("an admin was refused an admin route: %d", code)
	}
	if code := call(t, app.router, http.MethodGet, "/api/apikeys", member, nil).Code; code != http.StatusForbidden {
		t.Fatalf("a non-admin reached an admin route: %d", code)
	}
	if code := call(t, app.router, http.MethodPost, "/api/auth/logout", admin, nil).Code; code != http.StatusOK {
		t.Fatalf("logout = %d", code)
	}
	if code := call(t, app.router, http.MethodGet, "/api/auth/me", admin, nil).Code; code != http.StatusUnauthorized {
		t.Fatalf("the session survived the logout: %d", code)
	}
}

// Sign-up and password sign-in are gone: the routes do not exist, so there is
// no endpoint left to probe for an account.
func TestPasswordRoutesAreGone(t *testing.T) {
	app := newLiveApp(t)
	for _, path := range []string{"/api/auth/register", "/api/auth/login"} {
		body := map[string]string{"email": "a@facile.studio", "name": "A", "password": "a-long-enough-password"}
		if code := call(t, app.router, http.MethodPost, path, "", body).Code; code != http.StatusNotFound {
			t.Fatalf("POST %s = %d, want 404", path, code)
		}
	}
}

// Other apps ship logs with an ingest token and no session. That path must not
// depend on the sign-in rework: the legacy token and a per-app key both work,
// a missing or wrong token does not, and a dashboard session is not an ingest
// credential.
func TestIngestNeedsATokenAndNoSession(t *testing.T) {
	app := newLiveApp(t)
	_, key, err := apikeys.NewService(app.db).Create(context.Background(), apikeys.NewKey{App: "nuage"})
	if err != nil {
		t.Fatalf("create a key: %v", err)
	}
	_, session := app.signIn(t, "admin@facile.studio", true)
	entry := map[string]any{"app": "nuage", "level": "error", "message": "upload failed", "meta": map[string]any{"file_id": 42}}

	cases := []struct {
		name  string
		token string
		want  int
	}{
		{"legacy token", testIngestToken, http.StatusCreated},
		{"per-app key", key, http.StatusCreated},
		{"no token", "", http.StatusUnauthorized},
		{"wrong token", "nope", http.StatusUnauthorized},
		{"dashboard session", session, http.StatusUnauthorized},
	}
	for _, tc := range cases {
		if got := call(t, app.router, http.MethodPost, "/api/ingest", tc.token, entry).Code; got != tc.want {
			t.Errorf("%s: POST /api/ingest = %d, want %d", tc.name, got, tc.want)
		}
	}
	var stored int64
	app.db.Model(&schemas.LogEntry{}).Count(&stored)
	if stored != 2 {
		t.Fatalf("stored %d entries, want 2", stored)
	}
}

// The SSO callback's half of the upsert, without a provider: first account in
// becomes the admin, a second sign-in finds the same row by (provider,
// subject) even though the address changed in the identity provider, and an
// unverified address may never adopt an account that already exists.
func TestUpsertFromOIDCOwnsTheAccountRules(t *testing.T) {
	db := testdb.Migrated(t)
	users := auth.NewUserStore(sqlHandle(t, db))
	ctx := context.Background()

	userID, err := users.UpsertFromOIDC(ctx, claims("abc", "someone@facile.studio", "Someone", true))
	if err != nil {
		t.Fatalf("first sign-in: %v", err)
	}
	var user schemas.User
	if err := db.First(&user, userID).Error; err != nil {
		t.Fatalf("load: %v", err)
	}
	if !user.IsAdmin {
		t.Fatal("the first account to sign in did not become admin")
	}
	if user.PasswordHash != "" {
		t.Fatal("an SSO account was given a password hash")
	}
	link(t, db, userID, "abc")

	again, err := users.UpsertFromOIDC(ctx, claims("abc", "renamed@facile.studio", "Renamed", true))
	if err != nil {
		t.Fatalf("second sign-in: %v", err)
	}
	if again != userID {
		t.Fatalf("a renamed user got a second account: %d then %d", userID, again)
	}
	if err := db.First(&user, userID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if user.Email != "renamed@facile.studio" {
		t.Fatalf("the address the provider asserts did not win: %q", user.Email)
	}

	if _, err := users.UpsertFromOIDC(ctx, claims("other", "renamed@facile.studio", "Someone Else", false)); err == nil {
		t.Fatal("an unverified address adopted an existing account")
	}
}

// An account that predates the switch to Registre keeps its data: the first
// verified Facile sign-in with the same address links to the same row. An
// address that already belongs to another subject of the same issuer is never
// handed over, verified or not.
func TestAnExistingAccountLinksByVerifiedEmailOnly(t *testing.T) {
	db := testdb.Migrated(t)
	users := auth.NewUserStore(sqlHandle(t, db))
	ctx := context.Background()
	legacy := schemas.User{Email: "legacy@facile.studio", Name: "Legacy", PasswordHash: "x", IsAdmin: true}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	first := claims("owner", "legacy@facile.studio", "Legacy", true)
	first.Locale = "de"
	linked, err := users.UpsertFromOIDC(ctx, first)
	if err != nil || linked != legacy.ID {
		t.Fatalf("the legacy account was not linked: id %d, err %v", linked, err)
	}
	link(t, db, linked, "owner")

	var stored schemas.User
	if err := db.First(&stored, legacy.ID).Error; err != nil || !stored.IsAdmin || stored.Locale != "de" {
		t.Fatalf("the linked account lost its data or its locale: %+v, %v", stored, err)
	}
	if _, err := users.UpsertFromOIDC(ctx, claims("stranger", "legacy@facile.studio", "Stranger", true)); err == nil {
		t.Fatal("a second subject of the same issuer took over a linked account")
	}
	var count int64
	db.Model(&schemas.User{}).Count(&count)
	if count != 1 {
		t.Fatalf("%d accounts after the refusals, want 1", count)
	}
}

func link(t *testing.T, db *gorm.DB, userID int64, subject string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`,
		userID, "https://sso.test/", subject).Error; err != nil {
		t.Fatalf("link the identity: %v", err)
	}
}

func claims(subject, email, name string, verified bool) porte.Claims {
	return porte.Claims{
		Provider:      "https://sso.test/",
		Subject:       subject,
		Email:         email,
		Name:          name,
		EmailVerified: verified,
	}
}

func TestBrowserIngestStaysReadableCrossOrigin(t *testing.T) {
	app := newLiveApp(t)
	_, key, err := apikeys.NewService(app.db).Create(context.Background(), apikeys.NewKey{
		App: "nuage", Kind: schemas.KeyKindPublic, AllowedOrigins: []string{"https://nuage.facile.studio"}, DailyQuota: 1000,
	})
	if err != nil {
		t.Fatalf("create a browser key: %v", err)
	}
	body := `{"events":[{"level":"error","message":"boom"}]}`
	post := func(origin string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/api/ingest/browser?key="+key, bytes.NewReader([]byte(body)))
		request.Header.Set("Content-Type", "text/plain;charset=UTF-8")
		request.Header.Set("Origin", origin)
		recorder := httptest.NewRecorder()
		app.router.ServeHTTP(recorder, request)
		return recorder
	}

	allowed := post("https://nuage.facile.studio")
	if allowed.Code != http.StatusCreated && allowed.Code != http.StatusAccepted && allowed.Code != http.StatusOK {
		t.Fatalf("allowed origin = %d: %s", allowed.Code, allowed.Body.String())
	}
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://nuage.facile.studio" {
		t.Fatalf("Access-Control-Allow-Origin = %q, the SDK could not read its response", got)
	}
	for _, name := range []string{"Cross-Origin-Resource-Policy", "Cross-Origin-Embedder-Policy"} {
		if got := allowed.Header().Get(name); got != "" {
			t.Errorf("%s = %q would block a cross-origin report", name, got)
		}
	}
	if refused := post("https://evil.example"); refused.Code != http.StatusForbidden {
		t.Fatalf("unlisted origin = %d, want 403", refused.Code)
	}
}

func sqlHandle(t *testing.T, db *gorm.DB) *sql.DB {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	return sqlDB
}
