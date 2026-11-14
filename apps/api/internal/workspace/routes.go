package workspace

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	slugPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	prefixPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
)

// Route sends the instance workspace's entries whose app name starts with
// one of Prefixes to the workspace Slug, named Name.
type Route struct {
	Slug     string
	Name     string
	Prefixes []string
}

// ParseRoutes reads INGEST_ROUTES: comma-separated `slug:Name:prefix|prefix`
// entries, such as `acme:Acme:acme-|acme_`.
func ParseRoutes(raw string) ([]Route, error) {
	var routes []Route
	for entry := range strings.SplitSeq(raw, ",") {
		if entry = strings.TrimSpace(entry); entry == "" {
			continue
		}
		parts := strings.Split(entry, ":")
		if len(parts) != 3 || !slugPattern.MatchString(parts[0]) || strings.TrimSpace(parts[1]) == "" {
			return nil, fmt.Errorf("INGEST_ROUTES: %q is not slug:Name:prefix|prefix", entry)
		}
		route := Route{Slug: parts[0], Name: strings.TrimSpace(parts[1])}
		for prefix := range strings.SplitSeq(parts[2], "|") {
			if !prefixPattern.MatchString(prefix) {
				return nil, fmt.Errorf("INGEST_ROUTES: %q has an invalid app prefix %q", entry, prefix)
			}
			route.Prefixes = append(route.Prefixes, prefix)
		}
		routes = append(routes, route)
	}
	return routes, nil
}

type prefixRoute struct {
	prefix      string
	workspaceID int64
}

// Router picks the workspace an instance entry is written to by its app name.
// The zero Router routes nothing.
type Router struct {
	prefixes []prefixRoute
}

// Resolve returns the workspace an entry of app goes to: the longest matching
// route when the scope is routable, else workspaceID unchanged.
func (r Router) Resolve(workspaceID int64, routable bool, app string) int64 {
	if !routable {
		return workspaceID
	}
	for _, p := range r.prefixes {
		if strings.HasPrefix(app, p.prefix) {
			return p.workspaceID
		}
	}
	return workspaceID
}

// SyncRoutes makes sure every route's workspace exists, exempt from plan
// limits, and builds the Router over them. A route's workspace that has no
// member yet takes the instance workspace's members and roles; one that has
// members keeps them and only follows the configured name.
func (s *Service) SyncRoutes(ctx context.Context, routes []Route) (Router, error) {
	var router Router
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, route := range routes {
			id, err := ensureRouted(tx, route)
			if err != nil {
				return err
			}
			for _, prefix := range route.Prefixes {
				router.prefixes = append(router.prefixes, prefixRoute{prefix: prefix, workspaceID: id})
			}
		}
		return nil
	})
	slices.SortStableFunc(router.prefixes, func(a, b prefixRoute) int { return len(b.prefix) - len(a.prefix) })
	return router, err
}

func ensureRouted(tx *gorm.DB, route Route) (int64, error) {
	slug := route.Slug
	ws := schemas.Workspace{Name: route.Name, Slug: &slug, PlanExempt: true}
	created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&ws)
	if created.Error != nil {
		return 0, created.Error
	}
	if created.RowsAffected == 0 {
		if err := tx.Where("slug = ?", slug).Take(&ws).Error; err != nil {
			return 0, err
		}
		if err := tx.Model(&ws).Update("name", route.Name).Error; err != nil {
			return 0, err
		}
	}
	return ws.ID, tx.Exec(`INSERT INTO workspace_members (workspace_id, user_id, role)
		SELECT ?, user_id, role FROM workspace_members WHERE workspace_id = ?
		AND NOT EXISTS (SELECT 1 FROM workspace_members WHERE workspace_id = ?)`,
		ws.ID, schemas.InstanceWorkspaceID, ws.ID).Error
}
