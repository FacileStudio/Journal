package auth

import (
	"context"
	stderrors "errors"

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
type UserStore struct {
	orm *gorm.DB
}

func NewUserStore(orm *gorm.DB) *UserStore {
	return &UserStore{orm: orm}
}

var _ porte.UserStore = (*UserStore)(nil)

// UpsertFromOIDC matches on (provider, subject) first, falls back to a
// verified email, and creates the account when neither finds anything.
//
// The advisory lock is the one Register already takes: without it two people
// signing in for the first time at the same moment both count zero users and
// both become admin.
func (s *UserStore) UpsertFromOIDC(ctx context.Context, claims porte.Claims) (int64, error) {
	email := normalizeEmail(claims.Email)
	if !validEmail(email) {
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
			return refreshProfile(tx, linked, email, name)
		}

		var existing schemas.User
		err = tx.Where("email = ?", email).First(&existing).Error
		switch {
		case err == nil:
			// Matching an existing account on the address alone is an
			// account takeover primitive when the provider lets a user
			// claim any address without proving it.
			if !claims.EmailVerified {
				return errors.Conflict("an account with this email already exists and the identity provider did not verify the address")
			}
			userID = existing.ID
			return refreshProfile(tx, existing.ID, email, name)
		case !stderrors.Is(err, gorm.ErrRecordNotFound):
			return errors.Internal("failed to look up the account", err)
		}

		var count int64
		if err := tx.Model(&schemas.User{}).Count(&count).Error; err != nil {
			return errors.Internal("failed to count users", err)
		}
		// password_hash is NOT NULL and no password exists for an SSO
		// account. The empty string is not a valid Argon2id encoding, so
		// VerifyPassword refuses it for every input and there is no
		// password that signs this user in.
		user := schemas.User{Email: email, Name: name, PasswordHash: "", IsAdmin: count == 0}
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

// refreshProfile keeps the name and the address the provider asserts, which is
// what makes the identity provider the source of truth for both. An empty name
// is not written over one the user already has: a provider that stops sending
// the claim should not blank the dashboard.
func refreshProfile(tx *gorm.DB, userID int64, email, name string) error {
	updates := map[string]any{"email": email}
	if name != "" {
		updates["name"] = name
	}
	if err := tx.Model(&schemas.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return errors.Internal("failed to update the account", err)
	}
	return nil
}

// ConfigExtra is Journal's addition to GET /auth/config. porte serves sso_only
// and oidc_enabled there; allow_registration is Journal's own and the login
// screen has read it since before any of this.
func ConfigExtra(allowRegistration bool) func() map[string]any {
	return func() map[string]any {
		return map[string]any{"allow_registration": allowRegistration}
	}
}
