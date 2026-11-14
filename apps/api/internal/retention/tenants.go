package retention

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Tenants reports the retention in days each limited workspace's plan
// allows, keyed by workspace id.
type Tenants interface {
	Retention(ctx context.Context) (map[int64]int, error)
}

const (
	expireWorkspaceBatch = `DELETE FROM log_entries WHERE id IN (
		SELECT id FROM log_entries WHERE workspace_id = $1 AND created_at < $2 ORDER BY created_at LIMIT $3)`
	purgeBatch = `DELETE FROM log_entries WHERE id IN (
		SELECT id FROM log_entries WHERE workspace_id = $1 AND workspace_id <> 1 LIMIT $2)`
	pendingPurges = `SELECT workspace_id FROM workspace_purges ORDER BY deleted_at LIMIT 100`
	purgeDone     = `DELETE FROM workspace_purges WHERE workspace_id = $1`
)

// sweepTenants applies each limited workspace's own window, then deletes the
// entries of every workspace that was deleted since the last sweep, an
// erased account's personal workspace among them. It returns both counts.
func sweepTenants(ctx context.Context, db *sql.DB, tenants Tenants, now time.Time, batch int) (int64, int64, error) {
	windows, err := tenants.Retention(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("retention: read the workspace plans: %w", err)
	}
	var expired int64
	for workspaceID, days := range windows {
		cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
		n, err := deleteBatches(ctx, db, expireWorkspaceBatch, batch, workspaceID, cutoff)
		expired += n
		if err != nil {
			return expired, 0, fmt.Errorf("retention: expire workspace %d: %w", workspaceID, err)
		}
	}
	purged, err := purge(ctx, db, batch)
	return expired, purged, err
}

func purge(ctx context.Context, db *sql.DB, batch int) (int64, error) {
	rows, err := db.QueryContext(ctx, pendingPurges)
	if err != nil {
		return 0, fmt.Errorf("retention: list deleted workspaces: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	var total int64
	for _, id := range ids {
		n, err := deleteBatches(ctx, db, purgeBatch, batch, id)
		total += n
		if err != nil {
			return total, fmt.Errorf("retention: purge workspace %d: %w", id, err)
		}
		if _, err := db.ExecContext(ctx, purgeDone, id); err != nil {
			return total, err
		}
	}
	return total, nil
}
