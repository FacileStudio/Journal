package mcp

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/modules/alerts"
	"github.com/FacileStudio/Journal/apps/api/modules/auth"
	"github.com/FacileStudio/Journal/apps/api/modules/logs"
	"github.com/FacileStudio/Journal/apps/api/modules/queries"
	"github.com/FacileStudio/Journal/apps/api/modules/sourcemaps"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/mcpauth"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	"github.com/FacileStudio/tronc/mcp/mcptest"
	"github.com/go-chi/chi/v5"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
)

// The Registre subjects of the test people: an administrator and a member.
const (
	adminSub  = "admin-sub"
	memberSub = "member-sub"
)

// identities resolves each test subject to its user.
type identities struct {
	porte.IdentityStore
	users map[string]int64
}

// Find resolves a known subject.
func (s identities) Find(_ context.Context, provider, sub string) (porte.StoredIdentity, error) {
	id, ok := s.users[sub]
	if !ok {
		return porte.StoredIdentity{}, porte.ErrNotFound
	}
	return porte.StoredIdentity{UserID: id, Provider: provider, Subject: sub}, nil
}

// fixture is a running MCP endpoint over Journal's real services.
type fixture struct {
	db       *gorm.DB
	issuer   *mcptest.Issuer
	service  *Service
	resource string
	users    map[string]int64
	front    func(http.Handler) http.Handler
}

// newFixture starts the endpoint on any free port.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	return newFixtureOn(t, nil)
}

// newFixtureOn starts the endpoint on listener, any free port when nil.
func newFixtureOn(t *testing.T, listener net.Listener) *fixture {
	t.Helper()
	f := &fixture{db: testdb.Migrated(t), issuer: mcptest.NewIssuer(t), users: map[string]int64{}, front: func(h http.Handler) http.Handler { return h }}
	f.db.TranslateError = true
	f.seed(t)
	var handler http.Handler
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.front(handler).ServeHTTP(w, r) }))
	if listener != nil {
		srv.Listener = listener
	}
	srv.Start()
	t.Cleanup(srv.Close)
	f.resource = srv.URL + "/mcp"
	f.service = f.build(t, listener != nil)
	router := chi.NewRouter()
	RegisterRoutes(router, f.service)
	handler = router
	return f
}

// seed creates the administrator and the member.
func (f *fixture) seed(t *testing.T) {
	t.Helper()
	for sub, admin := range map[string]bool{adminSub: true, memberSub: false} {
		var id int64
		err := f.db.Raw(`INSERT INTO users (email, name, is_admin) VALUES (?, ?, ?) RETURNING id`, sub+"@example.test", sub, admin).Scan(&id).Error
		must(t, err)
		f.users[sub] = id
	}
}

// build makes the MCP service behind the fake Registre.
func (f *fixture) build(t *testing.T, unlimited bool) *Service {
	t.Helper()
	verifier, err := mcpauth.Verifier(context.Background(), mcpauth.Config{
		Issuer: f.issuer.URL, Resource: f.resource, ScopePrefix: "journal", Identities: identities{users: f.users},
	})
	must(t, err)
	sqlDB, err := f.db.DB()
	must(t, err)
	users, err := auth.NewUserStore(context.Background(), sqlDB)
	must(t, err)
	perMinute := 0
	if unlimited {
		perMinute = 1_000_000
	}
	service, err := NewService(Deps{
		Logs: logs.NewService(f.db), Stacks: sourcemaps.NewService(f.db), Queries: queries.NewService(f.db),
		Alerts: alerts.NewService(f.db), Accounts: users,
	}, Config{
		Resource: f.resource, Issuer: f.issuer.URL, Version: "test", StateKey: []byte("state-key"), Verifier: verifier,
		Pending: mcpkit.PostgresPending{DB: sqlDB}, PerMinute: perMinute,
	})
	must(t, err)
	return service
}

// token is the administrator's token with scopes, or both when none is given.
func (f *fixture) token(t testing.TB, scopes ...string) string {
	if len(scopes) == 0 {
		scopes = []string{"journal.read", "journal.write"}
	}
	return f.issuer.Token(t, f.resource, adminSub, scopes...)
}

// connect opens a session for subject with both scopes and no elicitation.
func (f *fixture) connect(t *testing.T, subject string) *sdk.ClientSession {
	t.Helper()
	return mcptest.Connect(t, f.resource, f.issuer.Token(t, f.resource, subject, "journal.read", "journal.write"), mcptest.ClientOptions{})
}

// entry stores a log entry age before now.
func (f *fixture) entry(t testing.TB, app, level, message string, age time.Duration, meta map[string]any) schemas.LogEntry {
	t.Helper()
	e := schemas.LogEntry{App: app, Level: level, Message: message, Meta: meta, CreatedAt: time.Now().Add(-age)}
	must(t, f.db.Create(&e).Error)
	return e
}

// query stores a saved query.
func (f *fixture) query(t testing.TB, name string) schemas.SavedQuery {
	t.Helper()
	q := schemas.SavedQuery{Name: name, Params: schemas.SavedQueryParams{App: "nuage", Levels: []string{"error"}}}
	must(t, f.db.Create(&q).Error)
	return q
}

// alert stores an enabled webhook alert rule on query.
func (f *fixture) alert(t testing.TB, name string, queryID int64) schemas.AlertRule {
	t.Helper()
	r := schemas.AlertRule{
		Name: name, SavedQueryID: queryID, Provider: schemas.AlertProviderWebhook, Threshold: 3, WindowMinutes: 5,
		WebhookURL: "https://hooks.example.test/services/T0/B0/secret-token", Enabled: true,
	}
	must(t, f.db.Create(&r).Error)
	return r
}

// count is how many rows of model match id.
func (f *fixture) count(model any, id int64) int {
	var n int64
	f.db.Model(model).Where("id = ?", id).Count(&n)
	return int(n)
}

// call runs a tool and decodes its result, failing on a tool error.
func call(t *testing.T, session *sdk.ClientSession, tool string, args map[string]any, out any) {
	t.Helper()
	result := mcptest.Call(t, session, tool, args)
	if result.IsError {
		t.Fatalf("%s: %s", tool, mcptest.Text(result))
	}
	mcptest.Decode(t, result, out)
}

// refused runs a tool and returns its error text, failing when it worked.
func refused(t *testing.T, session *sdk.ClientSession, tool string, args map[string]any) string {
	t.Helper()
	result := mcptest.Call(t, session, tool, args)
	if !result.IsError {
		t.Fatalf("%s must be refused, got %s", tool, mcptest.Text(result))
	}
	return mcptest.Text(result)
}

// actionID is the pending action id at the end of a confirm link.
func actionID(link string) string {
	return link[strings.LastIndex(link, "=")+1:]
}

// must fails the test on err.
func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
