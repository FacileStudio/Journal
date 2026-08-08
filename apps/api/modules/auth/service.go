package auth

import (
	"context"
	stderrors "errors"
	"net/mail"
	"strings"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/internal/authcrypto"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

const (
	minPasswordLen      = 12
	registrationLockKey = 0x6A6F75726E616C
)

type Service struct {
	orm        *gorm.DB
	sessions   porte.SessionStore
	sessionTTL time.Duration
}

// NewService takes the session store porte authenticates against, so a
// password login and an SSO login land in the same table and are ended by the
// same logout. There is one session model in this app and two ways to reach
// it, which is the whole point of adopting the kit.
func NewService(orm *gorm.DB, sessions porte.SessionStore, sessionTTL time.Duration) *Service {
	return &Service{orm: orm, sessions: sessions, sessionTTL: sessionTTL}
}

func (s *Service) Register(ctx context.Context, email, name, password string, allowRegistration bool) (*schemas.User, string, error) {
	email = normalizeEmail(email)
	if !validEmail(email) {
		return nil, "", errors.Invalid("a valid email is required")
	}
	if len(password) < minPasswordLen {
		return nil, "", errors.Invalid("password must be at least 12 characters")
	}

	hash, err := authcrypto.HashPassword(password)
	if err != nil {
		return nil, "", errors.Internal("failed to hash password", err)
	}

	user := schemas.User{Email: email, Name: strings.TrimSpace(name), PasswordHash: hash}

	txErr := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(registrationLockKey)).Error; err != nil {
			return errors.Internal("failed to acquire registration lock", err)
		}
		var count int64
		if err := tx.Model(&schemas.User{}).Count(&count).Error; err != nil {
			return errors.Internal("failed to count users", err)
		}
		if !allowRegistration && count > 0 {
			return errors.Forbidden("registration is disabled")
		}
		var existing int64
		if err := tx.Model(&schemas.User{}).Where("email = ?", email).Count(&existing).Error; err != nil {
			return errors.Internal("failed to check email", err)
		}
		if existing > 0 {
			return errors.Conflict("an account with this email already exists")
		}

		user.IsAdmin = count == 0
		if err := tx.Create(&user).Error; err != nil {
			if stderrors.Is(err, gorm.ErrDuplicatedKey) {
				return errors.Conflict("an account with this email already exists")
			}
			return errors.Internal("failed to create user", err)
		}
		return nil
	})
	if txErr != nil {
		return nil, "", txErr
	}

	token, err := s.issueSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	return &user, token, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*schemas.User, string, error) {
	email = normalizeEmail(email)

	var user schemas.User
	err := s.orm.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			authcrypto.EqualizeTiming(password)
			return nil, "", errors.Unauthorized("invalid email or password")
		}
		return nil, "", errors.Internal("failed to load user", err)
	}

	if !authcrypto.VerifyPassword(password, user.PasswordHash) {
		return nil, "", errors.Unauthorized("invalid email or password")
	}

	// Opportunistic and best effort, as it has always been: a sweep that
	// fails is a table that stays a little larger, not a login that fails.
	_, _ = s.sessions.DeleteExpired(ctx, time.Now())

	token, err := s.issueSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	return &user, token, nil
}

// IdentityForUser turns the user id porte authenticated into the identity the
// rest of Journal reads. porte hands the middleware a session and a user id
// and stops there: it holds no email and no is_admin, because what a role may
// do is the app's business and the profile lives in the app's own table.
//
// This is the query the old Authenticate already made as a join, moved one
// step later in the chain. It is unchanged in cost.
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
		// The session points at a user that no longer exists. porte's
		// foreign key cascades a delete, so this is a race rather than
		// a leak, and it is still not an authenticated request.
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

// issueSession mints a password session through porte's store, so it is the
// same row shape, the same hash and the same revocation as one issued by the
// SSO callback.
//
// The token still comes back to the client and still travels as a bearer:
// porte v0.1 sets its HttpOnly cookie on the OIDC callback only, and it does
// not export session issuance for an app that owns a local login. That is
// v0.2's job. Until then the password path keeps the transport it has, which
// regresses nothing, and both transports authenticate through porte.
func (s *Service) issueSession(ctx context.Context, userID int64) (string, error) {
	token, err := porte.NewToken()
	if err != nil {
		return "", errors.Internal("failed to generate token", err)
	}
	now := time.Now()
	_, err = s.sessions.Create(ctx, porte.Session{
		TokenHash:  porte.HashToken(token),
		UserID:     userID,
		CreatedAt:  now,
		LastUsedAt: now,
		ExpiresAt:  now.Add(s.sessionTTL),
	})
	if err != nil {
		return "", errors.Internal("failed to create session", err)
	}
	return token, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
