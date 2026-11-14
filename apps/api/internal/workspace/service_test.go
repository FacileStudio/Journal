package workspace_test

import (
	"context"
	"sync"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/Journal/apps/api/schemas"
)

// Concurrent first requests of a new account create exactly one personal
// workspace, owned by that account.
func TestFirstRequestsCreateOnePersonalWorkspace(t *testing.T) {
	db := testdb.Migrated(t)
	user := schemas.User{Email: "new@example.test", Name: "New"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	service := workspace.NewService(db)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Go(func() {
			_, err := service.Memberships(context.Background(), user.ID, user.Name)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var personal []schemas.Workspace
	if err := db.Where("personal_of = ?", user.ID).Find(&personal).Error; err != nil || len(personal) != 1 {
		t.Fatalf("want one personal workspace, got %d (%v)", len(personal), err)
	}
	memberships, err := service.Memberships(context.Background(), user.ID, user.Name)
	if err != nil || len(memberships) != 1 || memberships[0].Role != schemas.RoleOwner || memberships[0].ID == schemas.InstanceWorkspaceID {
		t.Fatalf("memberships = %+v %v", memberships, err)
	}
}

// INGEST_ROUTES entries parse into routes, the longest prefix wins, and only
// a routable scope is routed.
func TestRoutesParseAndResolve(t *testing.T) {
	routes, err := workspace.ParseRoutes("acme:Acme Corp:acme-|acme_, acmeweb:Acme Web:acme-web-")
	if err != nil || len(routes) != 2 || routes[0].Name != "Acme Corp" || len(routes[0].Prefixes) != 2 {
		t.Fatalf("parse = %+v %v", routes, err)
	}
	for _, bad := range []string{"acme", "Acme:Acme:acme-", "acme::acme-", "acme:Acme:", "acme:Acme:a b"} {
		if _, err := workspace.ParseRoutes(bad); err == nil {
			t.Fatalf("%q should be refused", bad)
		}
	}
	db := testdb.Migrated(t)
	router, err := workspace.NewService(db).SyncRoutes(context.Background(), routes)
	if err != nil {
		t.Fatal(err)
	}
	again, err := workspace.NewService(db).SyncRoutes(context.Background(), routes)
	if err != nil || again.Resolve(1, true, "acme-web-1") != router.Resolve(1, true, "acme-web-1") {
		t.Fatalf("a second sync must find the same workspaces: %v", err)
	}
	acme, web := router.Resolve(1, true, "acme-api"), router.Resolve(1, true, "acme-web-1")
	if acme == 1 || web == 1 || acme == web || router.Resolve(1, true, "sablier") != 1 || router.Resolve(1, false, "acme-api") != 1 {
		t.Fatalf("acme=%d web=%d", acme, web)
	}
}

// A Scope never matches outside its ids, and the zero Scope matches nothing.
func TestScopeSelection(t *testing.T) {
	access := workspace.Access{Memberships: []workspace.Membership{{ID: 1, Role: schemas.RoleMember}, {ID: 4, Role: schemas.RoleOwner}}}
	if _, err := access.Select([]int64{4, 9}); err == nil {
		t.Fatal("a foreign workspace was selected")
	}
	all, err := access.Select(nil)
	if err != nil || !all.Contains(1) || !all.Contains(4) || all.Contains(9) {
		t.Fatalf("all = %v %v", all.IDs(), err)
	}
	access.Scope = all
	if owned := access.Owned(); owned.Contains(1) || !owned.Contains(4) {
		t.Fatalf("owned = %v", owned.IDs())
	}
	if _, err := access.Target(1, true); err == nil {
		t.Fatal("a member wrote as an owner")
	}
	if target, err := access.Target(0, true); err != nil || target != 4 {
		t.Fatalf("default owner target = %d %v", target, err)
	}
	if _, err := access.Target(0, false); err == nil {
		t.Fatal("an ambiguous write target was guessed")
	}
	if !(workspace.Scope{}).Empty() {
		t.Fatal("the zero scope is not empty")
	}
}
