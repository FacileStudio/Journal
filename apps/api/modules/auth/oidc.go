package auth

import (
	"context"
	stderrors "errors"
	"net/mail"
	"strings"

	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// UserStore resolves an OIDC callback to a row in Journal's own users table.
//
// It is porte's escape hatch, taken deliberately. porte/pg ships a porte_users
// table and Journal could have moved onto it, but users carries is_admin —
// which porte transports no opinion about — and the first account created here
// becomes the administrator. That rule is product behaviour, it has to survive
// the switch to SSO, and it is precisely why porte asks the app to own this
// method instead of owning the write itself.
// UserStore reads and writes the users porte owns, and is the adapter Journal
// hands to the porte kit so its password and federated flows fit this database.
type UserStore struct {
	orm *gorm.DB
}

// NewUserStore wires a UserStore onto the database.
func NewUserStore(orm *gorm.DB) *UserStore {
	return &UserStore{orm: orm}
}

var (
	_ porte.UserStore         = (*UserStore)(nil)
	_ porte.PasswordUserStore = (*UserStore)(nil)
)

// UpsertFromOIDC matches on (provider, subject) first, falls back to a
// verified email, and creates the account when neither finds anything.
//
// The advisory lock is the one Register already takes: without it two people
// signing in for the first time at the same moment both count zero users and
// both become admin. Matching an existing account on the address alone is an
// account-takeover primitive when the provider lets a user claim any address
// without proving it, so an unverified claim is refused. An SSO account stores
// an empty password_hash — not a valid Argon2id encoding, so VerifyPassword
// refuses every input and no password can sign it in — and the first account
// ever created becomes admin.
func (s *UserStore) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if _, err := mail.ParseAddress(email); err != nil {
		return 0, errors.Invalid("the identity provider returned no usable email")
	}
	name := claims.DisplayName()

	var userID int64
	txErr := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(registrationLockKey)).Error; err != nil {
			return errors.Internal("failed to acquire registration lock", err)
		}

		var linked int64
		err := tx.Raw(
			`SELECT user_id FROM porte_identities WHERE provider = ? AND subject = ?`,
			claims.Provider, claims.Subject,
		).Scan(&linked).Error
		if err != nil {
			return errors.Internal("failed to resolve the identity", err)
		}
		if linked != 0 {
			userID = linked
			return refreshProfile(tx, linked, email, name, claims.AvatarURL)
		}

		var existing schemas.User
		err = tx.Where("email = ?", email).First(&existing).Error
		switch {
		case err == nil:
			if !claims.EmailVerified {
				return errors.Conflict("an account with this email already exists and the identity provider did not verify the address")
			}
			userID = existing.ID
			return refreshProfile(tx, existing.ID, email, name, claims.AvatarURL)
		case !stderrors.Is(err, gorm.ErrRecordNotFound):
			return errors.Internal("failed to look up the account", err)
		}

		var count int64
		if err := tx.Model(&schemas.User{}).Count(&count).Error; err != nil {
			return errors.Internal("failed to count users", err)
		}

		user := schemas.User{Email: email, Name: name, PasswordHash: "", IsAdmin: count == 0, AvatarURL: claims.AvatarURL}
		if err := tx.Create(&user).Error; err != nil {
			return errors.Internal("failed to create the account", err)
		}
		userID = user.ID
		return nil
	})
	if txErr != nil {
		return 0, txErr
	}
	return userID, nil
}

// refreshProfile keeps the name, the address and the avatar the provider
// asserts, which is what makes the identity provider the source of truth for
// all three. An empty value is not written over one the user already has: a
// provider that stops sending a claim, or an avatar fetch that failed, should
// not blank the dashboard.
func refreshProfile(tx *gorm.DB, userID int64, email, name, avatarURL string) error {
	updates := map[string]any{"email": email}
	if name != "" {
		updates["name"] = name
	}
	if avatarURL != "" {
		updates["avatar_url"] = avatarURL
	}
	if err := tx.Model(&schemas.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return errors.Internal("failed to update the account", err)
	}
	return nil
}

// CreateFromPassword creates a user row for a local registration. porte has
// already validated the address and hashed the password; what is left is the
// part porte has no opinion about, which is the whole of what a Journal user
// is: the first account created becomes the administrator.
//
// The advisory lock is the one Register has always taken. porte cannot take it
// — it does not own this database — which is exactly why this method is here.
func (s *UserStore) CreateFromPassword(ctx context.Context, email, name string) (int64, error) {
	var userID int64
	txErr := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(registrationLockKey)).Error; err != nil {
			return errors.Internal("failed to acquire registration lock", err)
		}
		var count int64
		if err := tx.Model(&schemas.User{}).Count(&count).Error; err != nil {
			return errors.Internal("failed to count users", err)
		}
		var existing int64
		if err := tx.Model(&schemas.User{}).Where("email = ?", email).Count(&existing).Error; err != nil {
			return errors.Internal("failed to check email", err)
		}
		if existing > 0 {
			return errors.Conflict("an account with this email already exists")
		}
		user := schemas.User{Email: email, Name: name, PasswordHash: "", IsAdmin: count == 0}
		if err := tx.Create(&user).Error; err != nil {
			if stderrors.Is(err, gorm.ErrDuplicatedKey) {
				return errors.Conflict("an account with this email already exists")
			}
			return errors.Internal("failed to create the account", err)
		}
		userID = user.ID
		return nil
	})
	if txErr != nil {
		return 0, txErr
	}
	return userID, nil
}

// FindByEmail returns the user id for an address, or porte.ErrNotFound.
func (s *UserStore) FindByEmail(ctx context.Context, email string) (int64, error) {
	var user schemas.User
	err := s.orm.WithContext(ctx).Select("id").Where("email = ?", email).First(&user).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return 0, porte.ErrNotFound
	}
	if err != nil {
		return 0, errors.Internal("failed to look up the account", err)
	}
	return user.ID, nil
}

// CountUsers backs porte/local's registration gate and the first-account rule.
func (s *UserStore) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	if err := s.orm.WithContext(ctx).Model(&schemas.User{}).Count(&count).Error; err != nil {
		return 0, errors.Internal("failed to count users", err)
	}
	return count, nil
}

// ConfigExtra is Journal's addition to GET /auth/config. porte serves sso_only
// and oidc_enabled there; allow_registration is Journal's own and the login
// screen has read it since before any of this.
func ConfigExtra(allowRegistration bool) func() map[string]any {
	return func() map[string]any {
		return map[string]any{"allow_registration": allowRegistration}
	}
}
