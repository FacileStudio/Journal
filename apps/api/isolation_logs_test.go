package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/FacileStudio/porte"
)

// B's log reads, every filter the dashboard, journal-cli search and the live
// tail poll send, return none of A's entries, and A's none of B's.
func TestIsolationLogSearchSeesOnlyOwnWorkspace(t *testing.T) {
	f := newTenants(t)
	since := url.QueryEscape(time.Now().Add(-time.Hour).UTC().Format(time.RFC3339))
	for _, path := range []string{"/api/logs", "/api/logs?app=sablier", "/api/logs?q=" + secretWord, "/api/logs?since=" + since, "/api/logs?level=info&limit=1000"} {
		if got := f.ids(t, path, f.b); overlaps(got, f.aEntries) {
			t.Fatalf("B read A's entries through %s: %v", path, got)
		}
	}
	if got := f.ids(t, "/api/logs", f.b); !slices.Equal(got, []int64{f.bEntry}) {
		t.Fatalf("B should read exactly its own entry, got %v", got)
	}
	if got := f.ids(t, "/api/logs", f.a); slices.Contains(got, f.bEntry) {
		t.Fatalf("A read B's entry: %v", got)
	}
}

// The browser's cookie and journal-cli's bearer token are the same session
// check, and both are scoped the same way.
func TestIsolationHoldsOnTheCookieTransport(t *testing.T) {
	f := newTenants(t)
	request := httptest.NewRequest(http.MethodGet, "/api/logs?q="+secretWord, nil)
	request.AddCookie(&http.Cookie{Name: porte.SessionCookieName, Value: f.b})
	recorder := httptest.NewRecorder()
	f.router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != `{"entries":[],"next_before":null}`+"\n" {
		t.Fatalf("cookie search as B = %d %s", recorder.Code, recorder.Body)
	}
}

// The histogram counts only the caller's workspaces.
func TestIsolationHistogramCountsOwnEntries(t *testing.T) {
	f := newTenants(t)
	var histogram struct {
		Buckets []struct{ Counts map[string]int64 } `json:"buckets"`
	}
	f.decode(t, f.call(t, http.MethodGet, "/api/logs/histogram", f.b, nil), http.StatusOK, &histogram)
	var total int64
	for _, bucket := range histogram.Buckets {
		for _, n := range bucket.Counts {
			total += n
		}
	}
	if total != 1 {
		t.Fatalf("B's histogram counts %d entries, want its 1", total)
	}
}

// Guessing ids gets nothing: every foreign id answers 404 on /context and
// /stack, exactly like an id that does not exist.
func TestIsolationEntryIdsCannotBeGuessed(t *testing.T) {
	f := newTenants(t)
	for _, id := range append(slices.Clone(f.aEntries), 999999) {
		for _, path := range []string{"/api/logs/%d/context", "/api/logs/%d/stack"} {
			if got := f.call(t, http.MethodGet, fmt.Sprintf(path, id), f.b, nil); got.Code != http.StatusNotFound {
				t.Fatalf("B on %s = %d %s, want 404", fmt.Sprintf(path, id), got.Code, got.Body)
			}
		}
	}
}

// Context around B's own entry stays in B's workspace even though A's
// entries sit on both sides of it in time.
func TestIsolationContextNeighboursStayInTheWorkspace(t *testing.T) {
	f := newTenants(t)
	got := f.ids(t, fmt.Sprintf("/api/logs/%d/context?before=50&after=50", f.bEntry), f.b)
	if !slices.Equal(got, []int64{f.bEntry}) {
		t.Fatalf("B's context should hold only its own entry, got %v", got)
	}
}

// The apps list is per workspace: B sees its own sablier, not A's, and not
// the acme apps the collector routes to another workspace.
func TestIsolationAppsListIsScoped(t *testing.T) {
	f := newTenants(t)
	var listed struct {
		Apps []struct {
			WorkspaceID int64  `json:"workspace_id"`
			Name        string `json:"name"`
			Count       int64  `json:"count"`
		} `json:"apps"`
	}
	f.decode(t, f.call(t, http.MethodGet, "/api/apps", f.b, nil), http.StatusOK, &listed)
	if len(listed.Apps) != 1 || listed.Apps[0].WorkspaceID != f.bWS || listed.Apps[0].Count != 1 {
		t.Fatalf("B's apps = %+v", listed.Apps)
	}
}

// Naming a workspace the caller does not belong to answers 404, alone or
// mixed with one of its own, and B's workspace list holds only its own.
func TestIsolationForeignWorkspaceSelectionIsNotFound(t *testing.T) {
	f := newTenants(t)
	for _, selection := range []string{"1", fmt.Sprintf("%d,1", f.bWS), fmt.Sprint(f.acmeWS), "abc"} {
		got := f.call(t, http.MethodGet, "/api/logs?workspace="+selection, f.b, nil)
		if got.Code != http.StatusNotFound && got.Code != http.StatusBadRequest {
			t.Fatalf("B selecting %s = %d %s", selection, got.Code, got.Body)
		}
	}
	f.personal(t, f.b)
}

// A reads several workspaces at once, or one, and a routed app lands only in
// its own workspace.
func TestIsolationMultiWorkspaceViewForAMember(t *testing.T) {
	f := newTenants(t)
	all := f.ids(t, "/api/logs", f.a)
	instance := f.ids(t, "/api/logs?workspace=1", f.a)
	acme := f.ids(t, fmt.Sprintf("/api/logs?workspace=%d", f.acmeWS), f.a)
	both := f.ids(t, fmt.Sprintf("/api/logs?workspace=1,%d", f.acmeWS), f.a)
	if len(acme) != 1 || overlaps(acme, instance) || len(both) != len(instance)+1 || len(all) != len(both) {
		t.Fatalf("all=%v instance=%v acme=%v both=%v", all, instance, acme, both)
	}
}
