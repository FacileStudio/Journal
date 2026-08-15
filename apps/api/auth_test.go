package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/env"
	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/go-chi/chi/v5"
)

// liveRouter is the real router over a real database, which is what it takes
// to say anything about the auth path: the credential is a row, the middleware
// is a query against it, and the identity the handlers read is a second query
// against a table porte does not know about. All three are database behaviour.
func liveRouter(t *testing.T) chi.Router {
	t.Helper()
	db := testdb.Migrated(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}

	appEnv := env.Config{AllowRegistration: true}
	kit, sessions, passwords, avatars := testKitFor(t, appEnv, db, sqlDB)
	return buildRouter(db, kit, sessions, passwords, avatars, appEnv, slog.New(slog.DiscardHandler))
}

func call(t *testing.T, router chi.Router, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		payload = bytes.NewReader(encoded)
	} else {
		payload = bytes.NewReader(nil)
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

// The whole point of the adoption: a password login now mints a porte session,
// porte's middleware authenticates it, Journal hydrates the profile porte does
// not carry, and porte's logout revokes it. Any one of those four failing
// leaves the dashboard unusable, and none of them can be checked without a
// database.
// TestPasswordLoginRunsThroughPorteEndToEnd exercises a full password login
// against the real middleware chain, and asserts that RequireAdmin actually
// reads is_admin out of the context Journal hydrates.
func TestPasswordLoginRunsThroughPorteEndToEnd(t *testing.T) {
	router := liveRouter(t)

	registered := call(t, router, http.MethodPost, "/api/auth/register", "", map[string]string{
		"email": "someone@facile.studio", "name": "Someone", "password": "a-long-enough-password",
	})
	if registered.Code != http.StatusCreated {
		t.Fatalf("register = %d: %s", registered.Code, registered.Body)
	}
	var created struct {
		Token string `json:"token"`
		User  struct {
			IsAdmin bool `json:"is_admin"`
		} `json:"user"`
	}
	if err := json.Unmarshal(registered.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode register: %v", err)
	}
	if !created.User.IsAdmin {
		t.Fatal("the first account did not become admin")
	}

	me := call(t, router, http.MethodGet, "/api/auth/me", created.Token, nil)
	if me.Code != http.StatusOK {
		t.Fatalf("me = %d: %s", me.Code, me.Body)
	}

	keys := call(t, router, http.MethodGet, "/api/apikeys", created.Token, nil)
	if keys.Code != http.StatusOK {
		t.Fatalf("an admin was refused an admin route: %d %s", keys.Code, keys.Body)
	}

	loggedOut := call(t, router, http.MethodPost, "/api/auth/logout", created.Token, nil)
	if loggedOut.Code != http.StatusOK {
		t.Fatalf("logout = %d: %s", loggedOut.Code, loggedOut.Body)
	}
	var logout struct {
		LoggedOut bool `json:"logged_out"`
	}
	if err := json.Unmarshal(loggedOut.Body.Bytes(), &logout); err != nil || !logout.LoggedOut {
		t.Fatalf("unexpected logout body: %s", loggedOut.Body)
	}

	if after := call(t, router, http.MethodGet, "/api/auth/me", created.Token, nil); after.Code != http.StatusUnauthorized {
		t.Fatalf("the session survived the logout: %d", after.Code)
	}
}

// A second account is not an admin, and the admin routes have to say so
// through the identity Journal hydrated rather than through anything porte
// decided — porte transports roles and assigns none.
func TestASecondAccountIsNotAnAdmin(t *testing.T) {
	router := liveRouter(t)

	for _, email := range []string{"first@facile.studio", "second@facile.studio"} {
		if code := call(t, router, http.MethodPost, "/api/auth/register", "", map[string]string{
			"email": email, "name": "Someone", "password": "a-long-enough-password",
		}).Code; code != http.StatusCreated {
			t.Fatalf("register %s = %d", email, code)
		}
	}

	signedIn := call(t, router, http.MethodPost, "/api/auth/login", "", map[string]string{
		"email": "second@facile.studio", "password": "a-long-enough-password",
	})
	if signedIn.Code != http.StatusOK {
		t.Fatalf("login = %d: %s", signedIn.Code, signedIn.Body)
	}
	var session struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(signedIn.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode login: %v", err)
	}

	if code := call(t, router, http.MethodGet, "/api/apikeys", session.Token, nil).Code; code != http.StatusForbidden {
		t.Fatalf("a non-admin reached an admin route: %d", code)
	}
}

// The SSO callback's half of the upsert, without a provider: first account in
// becomes the admin, a second sign-in finds the same row by (provider,
// subject) even though the address changed in the identity provider, and an
// unverified address may never adopt an account that already exists.
// TestUpsertFromOIDCOwnsTheAccountRules checks the upsert's edge rules: porte
// writes the identity row after the upsert returns, so the second sign-in is
// what reads it; and matching an existing account on an address the provider
// will not vouch for is an account-takeover primitive, so it is refused.
func TestUpsertFromOIDCOwnsTheAccountRules(t *testing.T) {
	db := testdb.Migrated(t)
	users := auth.NewUserStore(db)
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

	if err := db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`,
		userID, "https://sso.test/", "abc").Error; err != nil {
		t.Fatalf("link the identity: %v", err)
	}

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

func claims(subject, email, name string, verified bool) porte.Claims {
	return porte.Claims{
		Provider:      "https://sso.test/",
		Subject:       subject,
		Email:         email,
		Name:          name,
		EmailVerified: verified,
	}
}
