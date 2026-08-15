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
}

// NewService takes porte's local kit, which owns the passwords now: argon2id
// with the parameters this app already used, the constant-time compare, the
// equalised timing on an unknown address, and the session the credential lands
// in. What is left here is what porte has no opinion about — what a Journal
// user is, and who is an administrator.
func NewService(orm *gorm.DB, passwords *local.Kit) *Service {
	return &Service{orm: orm, local: passwords}
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
