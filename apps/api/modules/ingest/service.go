package ingest

import (
	"context"

	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

type Service struct {
	orm *gorm.DB
}

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

func (s *Service) Ingest(ctx context.Context, entries []schemas.LogEntry) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	if err := s.orm.WithContext(ctx).CreateInBatches(entries, 500).Error; err != nil {
		return 0, errors.Internal("failed to store log entries", err)
	}
	return len(entries), nil
}
