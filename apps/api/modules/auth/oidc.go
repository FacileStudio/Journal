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
// UserStore is the adapter Journal hands to the porte kit so the federated
// flow fits this database.
type UserStore struct {
	orm *gorm.DB
}

// NewUserStore wires a UserStore onto the database.
func NewUserStore(orm *gorm.DB) *UserStore {
	return &UserStore{orm: orm}
}

var _ porte.UserStore = (*UserStore)(nil)

// UpsertFromOIDC matches on (provider, subject) first, falls back to a
// verified email, and creates the account when neither finds anything.
//
// The advisory lock keeps two people signing in for the first time at the same
// moment from both counting zero users and both becoming admin. Matching an
// existing account on the address alone is an account-takeover primitive when
// the provider lets a user claim any address without proving it, so an
// unverified claim is refused. A verified address still does not link into an
// account that already holds another subject of this issuer: that would hand a
// recycled address to a stranger, so it is refused too. An SSO account stores
// an empty password_hash, and the first account ever created becomes admin.
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
			return refreshProfile(tx, linked, email, name, photoURL(claims.Picture), claims.Locale)
		}

		var existing schemas.User
		err = tx.Where("email = ?", email).First(&existing).Error
		switch {
		case err == nil:
			if !claims.EmailVerified {
				return errors.Conflict("an account with this email already exists and the identity provider did not verify the address")
			}
			if err := refuseOtherSubject(tx, existing.ID, claims); err != nil {
				return err
			}
			userID = existing.ID
			return refreshProfile(tx, existing.ID, email, name, photoURL(claims.Picture), claims.Locale)
		case !stderrors.Is(err, gorm.ErrRecordNotFound):
			return errors.Internal("failed to look up the account", err)
		}

		var count int64
		if err := tx.Model(&schemas.User{}).Count(&count).Error; err != nil {
			return errors.Internal("failed to count users", err)
		}

		user := schemas.User{Email: email, Name: name, PasswordHash: "", IsAdmin: count == 0, AvatarURL: photoURL(claims.Picture), Locale: localeOrDefault(claims.Locale)}
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
// all three. An empty name is not written over one the user already has, so a
// provider that stops sending the claim does not blank the dashboard. The
// avatar is always written: an empty one means the provider has no picture to
// offer, and keeping the old URL would keep showing a picture the user removed.
func refreshProfile(tx *gorm.DB, userID int64, email, name, avatarURL, locale string) error {
	updates := map[string]any{"email": email, "avatar_url": avatarURL}
	if name != "" {
		updates["name"] = name
	}
	if locale != "" {
		updates["locale"] = locale
	}
	if err := tx.Model(&schemas.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return errors.Internal("failed to update the account", err)
	}
	return nil
}

// refuseOtherSubject stops an address from linking into an account that already
// holds a different subject of the same issuer.
func refuseOtherSubject(tx *gorm.DB, userID int64, claims porte.Claims) error {
	var others int64
	err := tx.Raw(
		`SELECT count(*) FROM porte_identities WHERE user_id = ? AND provider = ? AND subject <> ?`,
		userID, claims.Provider, claims.Subject,
	).Scan(&others).Error
	if err != nil {
		return errors.Internal("failed to check the account's identities", err)
	}
	if others > 0 {
		return errors.Conflict("this account is already linked to another identity from the provider")
	}
	return nil
}

// localeOrDefault returns the normalised claim, or the default when the
// provider sent none.
func localeOrDefault(locale string) string {
	if locale == "" {
		return porte.DefaultLocale
	}
	return locale
}

// photoURL returns the picture claim when it is an https URL, and "" otherwise.
//
// Journal stores the URL the provider hands over and renders it, instead of
// downloading a copy at sign-in. Registre's picture claim is one stable URL per
// user that always serves the current picture, revalidated on every load, so a
// new picture shows on the next page load with no new sign-in. A local copy was
// refreshed only at sign-in, went stale the moment the user changed it, and
// fetching an attacker-influenced URL from inside the network was an SSRF
// surface. Anything that is not https (a data: placeholder, a plain-http host)
// is reported as no picture, and the dashboard draws the initial instead.
func photoURL(pictureClaim string) string {
	if strings.HasPrefix(strings.ToLower(pictureClaim), "https://") {
		return pictureClaim
	}
	return ""
}

// BackfillAvatars gives every account that has no avatar yet the picture URL
// of its identity at issuer, <issuer>/avatars/<subject>, without waiting for a
// sign-in.
//
// Accounts that signed in before Journal stored the picture claim hold an
// empty avatar_url until their next sign-in, which for a long session can be a
// month. Registre's picture URL is derived from the subject alone and answers
// for every user, so it can be written now. Only rows whose identity was issued
// by this issuer are touched, a non-empty avatar_url is never overwritten, and
// a non-https issuer writes nothing, which is the rule photoURL applies to the
// claim itself. Running it again changes nothing.
func (s *UserStore) BackfillAvatars(ctx context.Context, issuer string) (int64, error) {
	base := photoURL(strings.TrimSuffix(issuer, "/"))
	if base == "" {
		return 0, nil
	}
	result := s.orm.WithContext(ctx).Exec(
		`UPDATE users SET avatar_url = ? || '/avatars/' || porte_identities.subject
		   FROM porte_identities
		  WHERE porte_identities.user_id = users.id
		    AND porte_identities.provider = ?
		    AND users.avatar_url = ''`,
		base, issuer)
	if result.Error != nil {
		return 0, errors.Internal("failed to backfill avatars", result.Error)
	}
	return result.RowsAffected, nil
}
