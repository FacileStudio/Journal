package plans

import (
	"context"

	"github.com/FacileStudio/tronc/errors"
)

// Admit reserves n entries against workspaceID's daily cap before they are
// written, in one conditional upsert so two concurrent batches cannot both
// pass at the edge. The day is UTC.
func (s *Service) Admit(ctx context.Context, workspaceID int64, n int) error {
	if n <= 0 {
		return nil
	}
	limits, err := s.Limits(ctx, workspaceID)
	if err != nil {
		return err
	}
	if limits.Exempt || limits.EntriesDaily < 0 {
		return nil
	}
	if int64(n) > limits.EntriesDaily {
		return errors.RateLimited("this batch alone exceeds the workspace's daily entry limit")
	}
	result := s.orm.WithContext(ctx).Exec(`
		INSERT INTO workspace_usage (workspace_id, day, count)
		VALUES (?, (now() at time zone 'utc')::date, ?)
		ON CONFLICT (workspace_id, day) DO UPDATE
		   SET count = workspace_usage.count + excluded.count
		 WHERE workspace_usage.count + excluded.count <= ?`,
		workspaceID, n, limits.EntriesDaily)
	if result.Error != nil {
		return errors.Internal("failed to account for the daily entry limit", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.RateLimited("this workspace's daily entry limit is reached")
	}
	return nil
}

// Retention returns the retention in days of every workspace whose plan
// limits it, keyed by workspace id. Exempt and unlimited workspaces are left
// to the instance-wide policy.
func (s *Service) Retention(ctx context.Context) (map[int64]int, error) {
	var ids []int64
	if err := s.orm.WithContext(ctx).Table("workspaces").Where("NOT plan_exempt").Pluck("id", &ids).Error; err != nil {
		return nil, errors.Internal("failed to list the workspaces", err)
	}
	out := make(map[int64]int, len(ids))
	for _, id := range ids {
		limits, err := s.Limits(ctx, id)
		if err != nil {
			return nil, err
		}
		if !limits.Exempt && limits.RetentionDays > 0 {
			out[id] = int(limits.RetentionDays)
		}
	}
	return out, nil
}
