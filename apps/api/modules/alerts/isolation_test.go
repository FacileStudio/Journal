package alerts

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"gorm.io/gorm"
)

type hooks struct {
	mu    sync.Mutex
	paths []string
}

func (h *hooks) fired() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.paths...)
}

func watch(t *testing.T, db *gorm.DB, workspaceID int64, name, hook string) {
	t.Helper()
	query := schemas.SavedQuery{WorkspaceID: workspaceID, Name: name, Params: schemas.SavedQueryParams{App: "sablier"}}
	if err := db.Create(&query).Error; err != nil {
		t.Fatal(err)
	}
	rule := schemas.AlertRule{WorkspaceID: workspaceID, Name: name, SavedQueryID: query.ID, Provider: schemas.AlertProviderWebhook,
		Threshold: 1, WindowMinutes: 5, WebhookURL: hook + "/" + name, Enabled: true}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
}

// A rule counts its own workspace only: B's rule over the same app name does
// not fire on A's entries, and A's rule fires once on its own.
func TestEvaluatorCountsTheRulesWorkspaceOnly(t *testing.T) {
	db := testdb.Migrated(t)
	received := &hooks{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.mu.Lock()
		received.paths = append(received.paths, r.URL.Path)
		received.mu.Unlock()
	}))
	t.Cleanup(server.Close)
	tenant := schemas.Workspace{Name: "B"}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatal(err)
	}
	watch(t, db, 1, "a", server.URL)
	watch(t, db, tenant.ID, "b", server.URL)
	if err := db.Create(&schemas.LogEntry{WorkspaceID: 1, App: "sablier", Level: "error", Message: "boom", CreatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	d := delivery{guarded: server.Client(), trusted: server.Client()}
	evaluateDueRules(context.Background(), db, d, slog.New(slog.DiscardHandler))
	if got := received.fired(); len(got) != 1 || got[0] != "/a" {
		t.Fatalf("webhooks fired %v, want only A's", got)
	}
}

// A tenant's webhook always goes through the guarded client, even to a host
// the instance trusts; only the instance workspace reaches it unguarded.
func TestTrustedHostsAreForTheInstanceOnly(t *testing.T) {
	guarded, trusted := &http.Client{}, &http.Client{}
	d := delivery{guarded: guarded, trusted: trusted, allowedHosts: []string{"antenne.internal"}}
	if d.clientFor(2, "antenne.internal") != guarded || d.clientFor(1, "antenne.internal") != trusted || d.clientFor(1, "example.com") != guarded {
		t.Fatal("a tenant reached the trusted client, or the instance lost it")
	}
}
