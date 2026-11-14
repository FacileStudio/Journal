// Package retention keeps log_entries inside its storage policy. Entries
// older than the rolling window are deleted, which is the GDPR storage
// limitation (Article 5(1)(e)), and the table never holds more than a ceiling
// of entries, so one chatty producer cannot fill the disk inside the window.
//
// Both bounds delete in batches. A backlog of millions of rows, after the
// window is shortened or the ceiling lowered, never becomes one transaction
// holding every row lock and one burst of WAL, and a shutdown stops the job
// between two batches.
package retention

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	troncenv "github.com/FacileStudio/tronc/env"
)

const (
	// Interval is how often Run enforces the policy.
	Interval = time.Hour

	// BatchSize is the most rows one DELETE removes.
	BatchSize = 10000

	// DefaultDays is the rolling window when RETENTION_DAYS is unset.
	DefaultDays = 90

	// DefaultMaxEntries is the ceiling when RETENTION_MAX_ENTRIES is unset:
	// about 11 GB of table and indexes at the 1.1 KB an entry measured in
	// production on 2026-11-14, which is where docs/architecture.md moves
	// retention to partitions.
	DefaultMaxEntries = 10_000_000

	// CNILMaxDays is the upper end of the six months to one year CNIL
	// deliberation 2021-122 recommends for logs. A longer window needs a
	// documented justification.
	CNILMaxDays = 365
)

const (
	expireBatch = `DELETE FROM log_entries WHERE id IN (
		SELECT id FROM log_entries WHERE created_at < $1 ORDER BY created_at LIMIT $2)`
	trimBatch = `DELETE FROM log_entries WHERE id IN (
		SELECT id FROM log_entries WHERE id <= $1 ORDER BY id LIMIT $2)`
	newestID = `SELECT coalesce(max(id), 0) FROM log_entries`
)

// Policy is what the job enforces. A zero field disables that bound.
type Policy struct {
	Days       int
	MaxEntries int
}

// Result counts the entries one Sweep deleted, by the bound that removed them.
type Result struct {
	Expired  int64
	Overflow int64
}

// Load reads RETENTION_DAYS and RETENTION_MAX_ENTRIES and refuses a negative
// bound.
func Load() (Policy, error) {
	var p Policy
	var err error
	if p.Days, err = troncenv.Int("RETENTION_DAYS", DefaultDays); err != nil {
		return Policy{}, err
	}
	if p.MaxEntries, err = troncenv.Int("RETENTION_MAX_ENTRIES", DefaultMaxEntries); err != nil {
		return Policy{}, err
	}
	if p.Days < 0 || p.MaxEntries < 0 {
		return Policy{}, errors.New("RETENTION_DAYS and RETENTION_MAX_ENTRIES must be non-negative integers")
	}
	return p, nil
}

// Run enforces p at once and then every Interval until ctx ends. It warns
// about a window GDPR would ask to justify, and returns straight away when p
// bounds nothing.
func Run(ctx context.Context, db *sql.DB, p Policy, logger *slog.Logger) {
	warn(p, logger)
	if p.Days == 0 && p.MaxEntries == 0 {
		return
	}
	ticker := time.NewTicker(Interval)
	defer ticker.Stop()
	for {
		sweepAndLog(ctx, db, p, logger)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Sweep enforces p once: the rolling window first, then the ceiling.
func Sweep(ctx context.Context, db *sql.DB, p Policy) (Result, error) {
	return sweep(ctx, db, p, time.Now(), BatchSize)
}

func sweep(ctx context.Context, db *sql.DB, p Policy, now time.Time, batch int) (Result, error) {
	var r Result
	var err error
	if p.Days > 0 {
		cutoff := now.Add(-time.Duration(p.Days) * 24 * time.Hour)
		if r.Expired, err = deleteBatches(ctx, db, expireBatch, cutoff, batch); err != nil {
			return r, fmt.Errorf("retention: delete expired entries: %w", err)
		}
	}
	if p.MaxEntries > 0 {
		if r.Overflow, err = trim(ctx, db, p.MaxEntries, batch); err != nil {
			return r, fmt.Errorf("retention: trim to the ceiling: %w", err)
		}
	}
	return r, nil
}

// trim deletes every entry received before the newest ceiling. Ids come from
// one sequence in arrival order, so keeping id > max(id) - ceiling leaves at
// most ceiling rows for the price of one primary-key lookup, where counting
// rows would scan the whole table every hour.
func trim(ctx context.Context, db *sql.DB, ceiling int, batch int) (int64, error) {
	var newest int64
	if err := db.QueryRowContext(ctx, newestID).Scan(&newest); err != nil {
		return 0, err
	}
	bound := newest - int64(ceiling)
	if bound <= 0 {
		return 0, nil
	}
	return deleteBatches(ctx, db, trimBatch, bound, batch)
}

// deleteBatches runs query, a DELETE bounded by bound and a LIMIT of batch,
// until a batch comes back short, and returns the rows deleted.
func deleteBatches(ctx context.Context, db *sql.DB, query string, bound any, batch int) (int64, error) {
	var total int64
	for {
		res, err := db.ExecContext(ctx, query, bound, batch)
		if err != nil {
			return total, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, err
		}
		total += n
		if n < int64(batch) {
			return total, nil
		}
	}
}

func sweepAndLog(ctx context.Context, db *sql.DB, p Policy, logger *slog.Logger) {
	started := time.Now()
	r, err := Sweep(ctx, db, p)
	attrs := []any{
		slog.Int64("expired", r.Expired),
		slog.Int64("overflow", r.Overflow),
		slog.Int("retention_days", p.Days),
		slog.Int("max_entries", p.MaxEntries),
		slog.Duration("took", time.Since(started)),
	}
	switch {
	case err != nil && ctx.Err() == nil:
		logger.Error("retention sweep failed", append(attrs, slog.Any("error", err))...)
	case err == nil && r.Expired+r.Overflow > 0:
		logger.Info("retention deleted old log entries", attrs...)
	}
}

func warn(p Policy, logger *slog.Logger) {
	switch {
	case p.Days == 0:
		logger.Warn("RETENTION_DAYS is 0: log entries are kept forever, which needs a written justification under GDPR storage-limitation rules; set it to a positive number of days unless you have one")
	case p.Days > CNILMaxDays:
		logger.Warn("RETENTION_DAYS is over a year: CNIL deliberation 2021-122 recommends six months to a year for logs, and a longer window needs a documented justification", slog.Int("retention_days", p.Days))
	}
	if p.MaxEntries == 0 {
		logger.Warn("RETENTION_MAX_ENTRIES is 0: log_entries has no ceiling and grows with ingest volume until the window expires entries")
	}
}
