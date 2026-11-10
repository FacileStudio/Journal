package deletions

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"
	portepg "github.com/FacileStudio/porte/pg"
	"gorm.io/gorm"
)

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func count(t *testing.T, db *gorm.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

type fixture struct {
	db          *gorm.DB
	service     *Service
	gone, other int64
	issuer      string
	deleted     []directory.Deleted
}

func seedPerson(t *testing.T, db *gorm.DB, issuer, email, subject string, admin bool) int64 {
	t.Helper()
	user := schemas.User{Email: email, Name: email, IsAdmin: admin}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	tag := strconv.FormatInt(user.ID, 10)
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO porte_identities (user_id, provider, subject, refresh_token) VALUES (?, ?, ?, 'refresh')`, []any{user.ID, issuer, subject}},
		{`INSERT INTO porte_sessions (token_hash, user_id, label) VALUES (?, ?, 'cli')`, []any{"hash-" + tag, user.ID}},
		{`INSERT INTO porte_login_codes (code_hash, user_id, expires_at) VALUES (?, ?, now() + interval '1 hour')`, []any{"code-" + tag, user.ID}},
	}
	for _, statement := range statements {
		if err := db.Exec(statement.sql, statement.args...).Error; err != nil {
			t.Fatalf("seed credentials: %v", err)
		}
	}
	return user.ID
}

func seed(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{db: testdb.Migrated(t)}
	registre := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id, secret, ok := r.BasicAuth(); !ok || id != "journal" || secret != "s3cret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"deleted": f.deleted})
	}))
	t.Cleanup(registre.Close)
	f.issuer = registre.URL
	f.gone = seedPerson(t, f.db, f.issuer, "gone@facile.studio", "sub-gone", false)
	f.other = seedPerson(t, f.db, f.issuer, "other@facile.studio", "sub-other", true)
	if err := f.db.Create(&schemas.APIKey{App: "nuage", Kind: schemas.KeyKindSecret, Prefix: "p", KeyHash: "h"}).Error; err != nil {
		t.Fatalf("seed key: %v", err)
	}
	if err := f.db.Create(&schemas.LogEntry{App: "nuage", Level: "info", Message: "upload by gone@facile.studio"}).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}
	sqlDB, err := f.db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	client := directory.Client{Issuer: f.issuer, ClientID: "journal", ClientSecret: "s3cret"}
	users, err := auth.NewUserStore(context.Background(), sqlDB)
	if err != nil {
		t.Fatalf("user store: %v", err)
	}
	f.service = NewService(f.db, client, users.EraseAccount, portepg.New(sqlDB).EraseIdentity, quietLogger())
	return f
}

func (f *fixture) assertErased(t *testing.T) {
	t.Helper()
	checks := map[string]int64{
		`SELECT count(*) FROM users WHERE id = ?`:                  0,
		`SELECT count(*) FROM porte_identities WHERE user_id = ?`:  0,
		`SELECT count(*) FROM porte_sessions WHERE user_id = ?`:    0,
		`SELECT count(*) FROM porte_login_codes WHERE user_id = ?`: 0,
	}
	for query, want := range checks {
		if got := count(t, f.db, query, f.gone); got != want {
			t.Errorf("%s = %d, want %d", query, got, want)
		}
	}
	for _, query := range []string{
		`SELECT count(*) FROM users WHERE id = ?`,
		`SELECT count(*) FROM porte_identities WHERE user_id = ?`,
		`SELECT count(*) FROM porte_sessions WHERE user_id = ?`,
		`SELECT count(*) FROM porte_login_codes WHERE user_id = ?`,
	} {
		if got := count(t, f.db, query, f.other); got != 1 {
			t.Errorf("another person's data was touched: %s = %d", query, got)
		}
	}
	if got := count(t, f.db, `SELECT count(*) FROM api_keys`); got != 1 {
		t.Errorf("an instance-owned ingest key was removed: %d", got)
	}
}

// A deleted subject loses the account with its identity, sessions and login
// codes. Other people and instance-owned rows are untouched, and a second call
// changes nothing.
func TestErasePersonDeletesPersonalDataOnly(t *testing.T) {
	f := seed(t)
	for range 2 {
		if err := f.service.ErasePerson(context.Background(), f.issuer, "sub-gone"); err != nil {
			t.Fatalf("erase: %v", err)
		}
	}
	f.assertErased(t)
	if err := f.service.ErasePerson(context.Background(), f.issuer, "never-seen"); err != nil {
		t.Fatalf("an unknown subject must be a no-op: %v", err)
	}
}

// The full receiver: Registre lists the person, Poll erases them through
// OnPersonDeleted and porte's EraseIdentity, the state is saved in the database
// and a second Poll, from a fresh receiver that loads that state, is a no-op.
func TestPollErasesTheListedPersonOnce(t *testing.T) {
	f := seed(t)
	f.deleted = []directory.Deleted{{Subject: "sub-gone", DeletedAt: time.Now().Add(-time.Minute)}}

	if err := f.service.receiver.Poll(context.Background()); err != nil {
		t.Fatalf("poll: %v", err)
	}
	f.assertErased(t)
	if got := count(t, f.db, `SELECT count(*) FROM directory_deletion_state`); got != 1 {
		t.Fatalf("the receiver state was not persisted: %d rows", got)
	}

	seedPerson(t, f.db, f.issuer, "again@facile.studio", "sub-gone", false)
	restarted := NewService(f.db, f.service.receiver.Client, f.service.eraseAccount, f.service.receiver.Erase, quietLogger())
	if err := restarted.receiver.Poll(context.Background()); err != nil {
		t.Fatalf("second poll: %v", err)
	}
	if got := count(t, f.db, `SELECT count(*) FROM users WHERE email = 'again@facile.studio'`); got != 1 {
		t.Fatal("a handled subject reached the callback twice")
	}
}

func TestDeletionStateSurvivesARestart(t *testing.T) {
	st := state{orm: testdb.Migrated(t)}
	if saved, err := st.Load(context.Background()); err != nil || saved != nil {
		t.Fatalf("empty Load = %q, %v; want nil", saved, err)
	}
	for _, blob := range []string{"one", "two"} {
		if err := st.Save(context.Background(), []byte(blob)); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	if saved, err := st.Load(context.Background()); err != nil || string(saved) != "two" {
		t.Fatalf("Load = %q, %v; want two", saved, err)
	}
}

// Registre's push is public and unauthenticated by session, so a request with
// no valid signature must be refused.
func TestPushWithoutASignatureIsRefused(t *testing.T) {
	f := seed(t)
	request := httptest.NewRequest(http.MethodPost, directory.DeletionsPath, nil)
	recorder := httptest.NewRecorder()
	f.service.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unsigned push = %d, want 400", recorder.Code)
	}
	if got := count(t, f.db, `SELECT count(*) FROM users`); got != 2 {
		t.Fatalf("an unsigned push erased someone: %d users", got)
	}
}

// Every table with a foreign key to users is one the erasure test seeds and
// checks. A new table that references a person fails here until EraseData
// and assertErased cover it.
func TestErasureCoversEveryTableThatReferencesAUser(t *testing.T) {
	db := testdb.Migrated(t)
	var tables []string
	err := db.Raw(`SELECT DISTINCT conrelid::regclass::text FROM pg_constraint
		WHERE contype = 'f' AND confrelid = 'users'::regclass ORDER BY 1`).Scan(&tables).Error
	if err != nil {
		t.Fatalf("list the foreign keys: %v", err)
	}
	want := []string{"porte_identities", "porte_login_codes", "porte_sessions"}
	if !slices.Equal(tables, want) {
		t.Fatalf("tables referencing users = %v, want %v: erase the new one and assert it", tables, want)
	}
}
