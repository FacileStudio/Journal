package queries

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

var validLevels = map[string]bool{"debug": true, "info": true, "warn": true, "error": true}

// Service stores and lists saved queries.
type Service struct {
	orm *gorm.DB
}

// NewService wires a saved-query service onto the database.
func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

// List returns the saved queries of the workspaces in scope, by name.
func (s *Service) List(ctx context.Context, scope workspace.Scope) ([]schemas.SavedQuery, error) {
	var records []schemas.SavedQuery
	if err := scope.Where(s.orm.WithContext(ctx)).Order("name asc, workspace_id asc").Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list saved queries", err)
	}
	return records, nil
}

// Get returns the saved query id when it belongs to a workspace in scope.
func (s *Service) Get(ctx context.Context, scope workspace.Scope, id int64) (schemas.SavedQuery, error) {
	var record schemas.SavedQuery
	if err := scope.Where(s.orm.WithContext(ctx)).Where("id = ?", id).Take(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return schemas.SavedQuery{}, errors.NotFound("saved query not found")
		}
		return schemas.SavedQuery{}, errors.Internal("failed to load the saved query", err)
	}
	return record, nil
}

// Create stores a saved query in workspaceID. Names are unique per workspace.
func (s *Service) Create(ctx context.Context, workspaceID int64, name string, params schemas.SavedQueryParams) (*schemas.SavedQuery, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.Invalid("name is required")
	}
	if err := validateParams(params); err != nil {
		return nil, err
	}

	record := schemas.SavedQuery{WorkspaceID: workspaceID, Name: name, Params: params}
	if err := s.orm.WithContext(ctx).Create(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errors.Conflict("a saved query with this name already exists")
		}
		return nil, errors.Internal("failed to store saved query", err)
	}
	return &record, nil
}

// Delete removes the saved query id of a workspace in scope, unless an
// alert rule still watches it.
func (s *Service) Delete(ctx context.Context, scope workspace.Scope, id int64) error {
	if _, err := s.Get(ctx, scope, id); err != nil {
		return err
	}
	var dependents int64
	if err := s.orm.WithContext(ctx).Model(&schemas.AlertRule{}).Where("saved_query_id = ?", id).Count(&dependents).Error; err != nil {
		return errors.Internal("failed to check dependent alert rules", err)
	}
	if dependents > 0 {
		return errors.Conflict("delete dependent alert rules first")
	}
	if err := scope.Where(s.orm.WithContext(ctx)).Where("id = ?", id).Delete(&schemas.SavedQuery{}).Error; err != nil {
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
