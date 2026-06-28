package ingest

import (
	"context"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
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
