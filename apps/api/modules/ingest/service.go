package ingest

import (
	"context"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Admission charges a workspace's daily entry cap before entries are written.
type Admission interface {
	Admit(ctx context.Context, workspaceID int64, n int) error
}

// Service stores log entries and enforces the per-key ingest quota. router
// places the legacy token's entries in their workspace, and admission charges
// each workspace's plan; both are optional.
type Service struct {
	orm       *gorm.DB
	router    workspace.Router
	admission Admission
}

// Use sets how entries are routed to workspaces and how their daily cap is
// charged.
func (s *Service) Use(router workspace.Router, admission Admission) {
	s.router, s.admission = router, admission
}

// Place returns the workspace an entry of app goes to under scope.
func (s *Service) Place(scope authcontext.IngestScope, app string) int64 {
	return s.router.Resolve(scope.WorkspaceID, scope.Routable, app)
}

// NewService wires an ingest service onto the database.
func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

// ConsumeQuota reserves n entries against a key's daily allowance and reports
// whether they may be written.
//
// It reserves before the write rather than counting after it, so the worst
// case of a lost race is a handful of entries refused, not a quota discovered
// to have been blown an hour ago. One statement does the whole thing: the
// conditional upsert can only succeed if the new total still fits, which means
// two concurrent requests cannot both read "just under the limit" and both
// pass. The day is UTC because the quota is documented in UTC — CURRENT_DATE
// would silently follow the database's timezone instead.
func (s *Service) ConsumeQuota(ctx context.Context, keyID int64, quota, n int) error {
	if quota <= 0 || n <= 0 {
		return nil
	}
	if n > quota {
		return errors.RateLimited("this batch alone exceeds the key's daily quota")
	}

	result := s.orm.WithContext(ctx).Exec(`
		INSERT INTO api_key_usage (api_key_id, day, count)
		VALUES (?, (now() at time zone 'utc')::date, ?)
		ON CONFLICT (api_key_id, day) DO UPDATE
		   SET count = api_key_usage.count + excluded.count
		 WHERE api_key_usage.count + excluded.count <= ?`,
		keyID, n, quota)
	if result.Error != nil {
		return errors.Internal("failed to account for the daily quota", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.RateLimited("this key's daily quota is exhausted")
	}
	return nil
}

// Ingest charges every workspace the entries go to, then writes them. A
// workspace over its daily cap refuses the whole batch, before anything is
// written. An entry with no workspace is a programming error and refuses it too.
func (s *Service) Ingest(ctx context.Context, entries []schemas.LogEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	perWorkspace := map[int64]int{}
	for _, entry := range entries {
		if entry.WorkspaceID <= 0 {
			return 0, errors.Unauthorized("missing ingest scope")
		}
		perWorkspace[entry.WorkspaceID]++
	}
	if s.admission != nil {
		for workspaceID, n := range perWorkspace {
			if err := s.admission.Admit(ctx, workspaceID, n); err != nil {
				return 0, err
			}
		}
	}
	if err := s.orm.WithContext(ctx).CreateInBatches(entries, 500).Error; err != nil {
		return 0, errors.Internal("failed to store log entries", err)
	}
	return len(entries), nil
}
