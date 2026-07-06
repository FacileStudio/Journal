package logs

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/internal/logfilter"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

type ListParams struct {
	logfilter.Params
	Limit    int
	BeforeTs *time.Time
	BeforeID *int64
}

func (s *Service) filtered(ctx context.Context, params ListParams) *gorm.DB {
	return logfilter.Apply(s.orm.WithContext(ctx).Model(&schemas.LogEntry{}), params.Params)
}

func (s *Service) List(ctx context.Context, params ListParams) ([]schemas.LogEntry, error) {
	if params.Limit <= 0 {
		params.Limit = 100
	}
	if params.Limit > 1000 {
		params.Limit = 1000
	}

	query := s.filtered(ctx, params)
	if params.BeforeTs != nil && params.BeforeID != nil {
		query = query.Where("(created_at, id) < (?, ?)", *params.BeforeTs, *params.BeforeID)
	}

	var records []schemas.LogEntry
	if err := query.Order("created_at desc, id desc").Limit(params.Limit).Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list log entries", err)
	}
	return records, nil
}

type histogramRow struct {
	BucketTs time.Time
	Level    string
	Count    int64
}

func (s *Service) Histogram(ctx context.Context, params ListParams, bucketSeconds int64) ([]HistogramBucket, error) {
	var rows []histogramRow
	err := s.filtered(ctx, params).
		Select("to_timestamp(floor(extract(epoch from created_at) / ?) * ?) as bucket_ts, level, count(*) as count", bucketSeconds, bucketSeconds).
		Group("bucket_ts, level").
		Order("bucket_ts asc").
		Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to build histogram", err)
	}

	buckets := make([]HistogramBucket, 0, len(rows))
	for _, row := range rows {
		ts := row.BucketTs.UTC().Format(time.RFC3339)
		if len(buckets) == 0 || buckets[len(buckets)-1].Ts != ts {
			buckets = append(buckets, HistogramBucket{Ts: ts, Counts: map[string]int64{}})
		}
		buckets[len(buckets)-1].Counts[row.Level] = row.Count
	}
	return buckets, nil
}

func (s *Service) Context(ctx context.Context, id int64, before, after int) ([]schemas.LogEntry, error) {
	var anchor schemas.LogEntry
	if err := s.orm.WithContext(ctx).First(&anchor, id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("log entry not found")
		}
		return nil, errors.Internal("failed to load log entry", err)
	}

	var older []schemas.LogEntry
	if err := s.orm.WithContext(ctx).Model(&schemas.LogEntry{}).
		Where("(created_at, id) < (?, ?)", anchor.CreatedAt, anchor.ID).
		Order("created_at desc, id desc").
		Limit(before).
		Find(&older).Error; err != nil {
		return nil, errors.Internal("failed to load context entries", err)
	}

	var newer []schemas.LogEntry
	if err := s.orm.WithContext(ctx).Model(&schemas.LogEntry{}).
		Where("(created_at, id) > (?, ?)", anchor.CreatedAt, anchor.ID).
		Order("created_at asc, id asc").
		Limit(after).
		Find(&newer).Error; err != nil {
		return nil, errors.Internal("failed to load context entries", err)
	}

	merged := make([]schemas.LogEntry, 0, len(older)+len(newer)+1)
	for i := len(newer) - 1; i >= 0; i-- {
		merged = append(merged, newer[i])
	}
	merged = append(merged, anchor)
	merged = append(merged, older...)
	return merged, nil
}

type appRow struct {
	Name     string
	Count    int64
	LastSeen time.Time
}

func (s *Service) Apps(ctx context.Context) ([]AppSummary, error) {
	var rows []appRow
	if err := s.orm.WithContext(ctx).Model(&schemas.LogEntry{}).
		Select("app as name, count(*) as count, max(created_at) as last_seen").
		Group("app").
		Order("last_seen desc").
		Scan(&rows).Error; err != nil {
		return nil, errors.Internal("failed to list apps", err)
	}

	apps := make([]AppSummary, 0, len(rows))
	for _, row := range rows {
		apps = append(apps, AppSummary{
			Name:     row.Name,
			Count:    row.Count,
			LastSeen: row.LastSeen.UTC().Format(time.RFC3339),
		})
	}
	return apps, nil
}
