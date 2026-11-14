package workspace

import (
	"context"
	"strings"

	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

const personalLockSpace = 724300131

// Membership is one workspace the caller belongs to, and the caller's role in it.
type Membership struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Personal   bool   `json:"personal"`
	PlanExempt bool   `json:"plan_exempt"`
}

// Service reads and creates workspaces and their members.
type Service struct {
	orm *gorm.DB
}

// NewService wires the workspace service onto the database.
func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

// Memberships returns every workspace userID belongs to, creating the
// account's personal workspace first when it belongs to none. The creation
// runs under a transaction-scoped advisory lock on the user id, so two first
// requests racing each other create one workspace, not two.
func (s *Service) Memberships(ctx context.Context, userID int64, name string) ([]Membership, error) {
	found, err := s.list(ctx, s.orm, userID)
	if err != nil || len(found) > 0 {
		return found, err
	}
	err = s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(?, ?)`, personalLockSpace, int32(userID)).Error; err != nil {
			return err
		}
		if found, err = s.list(ctx, tx, userID); err != nil || len(found) > 0 {
			return err
		}
		return createPersonal(tx, userID, name)
	})
	if err != nil {
		return nil, errors.Internal("failed to prepare the personal workspace", err)
	}
	if len(found) > 0 {
		return found, nil
	}
	return s.list(ctx, s.orm, userID)
}

func (s *Service) list(ctx context.Context, db *gorm.DB, userID int64) ([]Membership, error) {
	var rows []struct {
		ID         int64
		Name       string
		Role       string
		PersonalOf *int64
		PlanExempt bool
	}
	err := db.WithContext(ctx).Table("workspace_members AS m").
		Select("w.id, w.name, m.role, w.personal_of, w.plan_exempt").
		Joins("JOIN workspaces w ON w.id = m.workspace_id").
		Where("m.user_id = ?", userID).Order("w.id").Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list workspaces", err)
	}
	out := make([]Membership, 0, len(rows))
	for _, row := range rows {
		out = append(out, Membership{
			ID: row.ID, Name: row.Name, Role: row.Role, Personal: row.PersonalOf != nil, PlanExempt: row.PlanExempt,
		})
	}
	return out, nil
}

func createPersonal(tx *gorm.DB, userID int64, name string) error {
	workspace := schemas.Workspace{Name: personalName(name), PersonalOf: &userID}
	if err := tx.Create(&workspace).Error; err != nil {
		return err
	}
	return tx.Create(&schemas.WorkspaceMember{WorkspaceID: workspace.ID, UserID: userID, Role: schemas.RoleOwner}).Error
}

func personalName(name string) string {
	if name = strings.TrimSpace(name); name != "" {
		return name
	}
	return "Personal"
}
