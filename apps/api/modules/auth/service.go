package auth

import (
	"context"
	stderrors "errors"
	"net/http"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// registrationLockKey is the advisory lock the account rules are taken under.
// porte cannot take it — it does not own this database — which is why the
// first-account rule lives in this package's PasswordUserStore.
const registrationLockKey = 0x6A6F75726E616C

// Service exposes login, logout and the current-user endpoint, delegating
// password work to the wrapped porte local kit.
type Service struct {
	orm   *gorm.DB
	local *local.Kit
	// removeAvatar deletes the locally cached avatar file an account points
	// at, and must already have decided that the URL belongs to this app's
	// store — erasure must not refuse because somebody's profile picture
	// lives on another host.
	removeAvatar func(ctx context.Context, avatarURL string) error
}

// NewService takes porte's local kit, which owns the passwords now: argon2id
// with the parameters this app already used, the constant-time compare, the
// equalised timing on an unknown address, and the session the credential lands
// in. What is left here is what porte has no opinion about — what a Journal
// user is, and who is an administrator.
func NewService(orm *gorm.DB, passwords *local.Kit, removeAvatar func(ctx context.Context, avatarURL string) error) *Service {
	return &Service{orm: orm, local: passwords, removeAvatar: removeAvatar}
}

// Register creates an account and signs it in. The token comes back for the
// bearer transport and the session cookie is set on the way out, so the same
// call serves the dashboard and a script.
func (s *Service) Register(ctx context.Context, w http.ResponseWriter, r *http.Request, email, name, password string) (*schemas.User, string, error) {
	userID, token, err := s.local.Register(ctx, w, r, email, name, password)
	if err != nil {
		return nil, "", err
	}
	user, err := s.UserByID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *Service) Login(ctx context.Context, w http.ResponseWriter, r *http.Request, email, password string) (*schemas.User, string, error) {
	userID, token, err := s.local.Login(ctx, w, r, email, password)
	if err != nil {
		return nil, "", err
	}
	user, err := s.UserByID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
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
// alike, so the token that made the request dies with the delete. A locally
// cached avatar file goes with it, removed before the row so a failed removal
// refuses the whole operation instead of leaving one behind. The log entries
// stay: they are keyed by app, never by user, so there is nothing of this
// person in them to erase.
//
// The delete itself carries the last-administrator rule inside its WHERE
// clause rather than checking first, so two administrators deleting in the
// same instant cannot race each other down to zero admins. A zero-affected-rows
// result then means either a missing account or the last admin, and the
// follow-up read tells those apart.
func (s *Service) DeleteAccount(ctx context.Context, userID int64) error {
	if s.removeAvatar != nil {
		var user schemas.User
		err := s.orm.WithContext(ctx).Select("avatar_url").First(&user, userID).Error
		if err == nil && user.AvatarURL != "" {
			if err := s.removeAvatar(ctx, user.AvatarURL); err != nil {
				return errors.Internal("failed to remove the avatar", err)
			}
		} else if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Internal("failed to load the account", err)
		}
	}

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
