package queries

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/gorm"
)

var validLevels = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

func (s *Service) List(ctx context.Context) ([]schemas.SavedQuery, error) {
	var records []schemas.SavedQuery
	if err := s.orm.WithContext(ctx).Order("name asc").Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list saved queries", err)
	}
	return records, nil
}

func (s *Service) Create(ctx context.Context, name string, params schemas.SavedQueryParams) (*schemas.SavedQuery, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.Invalid("name is required")
	}
	if err := validateParams(params); err != nil {
		return nil, err
	}

	record := schemas.SavedQuery{Name: name, Params: params}
	if err := s.orm.WithContext(ctx).Create(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.Conflict("a saved query with this name already exists")
		}
		return nil, errors.Internal("failed to store saved query", err)
	}
	return &record, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	var dependents int64
	if err := s.orm.WithContext(ctx).Model(&schemas.AlertRule{}).Where("saved_query_id = ?", id).Count(&dependents).Error; err != nil {
		return errors.Internal("failed to check dependent alert rules", err)
	}
	if dependents > 0 {
		return errors.Conflict("delete dependent alert rules first")
	}
	if err := s.orm.WithContext(ctx).Delete(&schemas.SavedQuery{}, id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrForeignKeyViolated) {
			return errors.Conflict("delete dependent alert rules first")
		}
		return errors.Internal("failed to delete saved query", err)
	}
	return nil
}

func validateParams(params schemas.SavedQueryParams) error {
	for _, level := range params.Levels {
		if !validLevels[level] {
			return errors.Invalid("levels must be a subset of debug, info, warn, error")
		}
	}
	return nil
}
