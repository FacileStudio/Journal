package auth

import (
	"context"
	stderrors "errors"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// registrationLockKey is the advisory lock the account rules are taken under.
// porte cannot take it — it does not own this database — which is why the
// first-account rule lives in this package's UserStore.
const registrationLockKey = 0x6A6F75726E616C

// Service exposes the current-user endpoint and account erasure.
type Service struct {
	orm *gorm.DB
}

// NewService wires the service onto the database. Sign-in is porte's OIDC kit;
// what is left here is what porte has no opinion about: what a Journal user is,
// and who is an administrator.
func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

// IdentityForUser turns the user id porte authenticated into the identity the
// rest of Journal reads. porte hands the middleware a session and a user id
// and stops there: it holds no email and no is_admin, because what a role may
// do is the app's business and the profile lives in the app's own table. A
// session pointing at a user that no longer exists — the foreign key cascades
// a delete, so this is a race rather than a leak — is still not an
// authenticated request and is refused.
func (s *Service) IdentityForUser(ctx context.Context, userID int64) (authcontext.Identity, error) {
	var out struct {
		ID      int64
		Email   string
		IsAdmin bool
	}
	err := s.orm.WithContext(ctx).
		Model(&schemas.User{}).
		Select("id", "email", "is_admin").
		Where("id = ?", userID).
		Scan(&out).Error
	if err != nil {
		return authcontext.Identity{}, errors.Internal("failed to load the account", err)
	}
	if out.ID == 0 {
		return authcontext.Identity{}, errors.Unauthorized("invalid auth token")
	}
	return authcontext.Identity{UserID: out.ID, Email: out.Email, IsAdmin: out.IsAdmin}, nil
}

// DeleteAccount erases the caller's account: one row, whose foreign keys
// cascade into every credential porte holds for it — identities and sessions
// alike, so the token that made the request dies with the delete. The log entries
// stay: they are keyed by app, never by user, and age out with RETENTION_DAYS.
//
// The delete itself carries the last-administrator rule inside its WHERE
// clause rather than checking first, so two administrators deleting in the
// same instant cannot race each other down to zero admins. A zero-affected-rows
// result then means either a missing account or the last admin, and the
// follow-up read tells those apart.
func (s *Service) DeleteAccount(ctx context.Context, userID int64) error {
	result := s.orm.WithContext(ctx).Exec(
		`DELETE FROM users
		 WHERE id = ? AND (NOT is_admin OR EXISTS (
		     SELECT 1 FROM users other WHERE other.is_admin AND other.id <> users.id))`,
		userID)
	if result.Error != nil {
		return errors.Internal("failed to delete the account", result.Error)
	}
	if result.RowsAffected == 1 {
		return nil
	}

	var remaining int64
	if err := s.orm.WithContext(ctx).Model(&schemas.User{}).
		Where("id = ?", userID).Count(&remaining).Error; err != nil {
		return errors.Internal("failed to load the account", err)
	}
	if remaining == 0 {
		return errors.NotFound("user not found")
	}
	return errors.Failed("the last administrator cannot delete their account; promote another admin first")
}

func (s *Service) UserByID(ctx context.Context, id int64) (*schemas.User, error) {
	var user schemas.User
	if err := s.orm.WithContext(ctx).First(&user, id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to load user", err)
	}
	return &user, nil
}
