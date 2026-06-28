package auth

import (
	"context"
	stderrors "errors"
	"net/mail"
	"strings"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/internal/authcrypto"
	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/gorm"
)

const (
	sessionTTL     = 30 * 24 * time.Hour
	minPasswordLen = 12
)

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

func (s *Service) HasUsers(ctx context.Context) (bool, error) {
	var count int64
	if err := s.orm.WithContext(ctx).Model(&schemas.User{}).Count(&count).Error; err != nil {
		return false, errors.Internal("failed to count users", err)
	}
	return count > 0, nil
}

func (s *Service) Register(ctx context.Context, email, name, password string) (*schemas.User, string, error) {
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

		user.IsAdmin = count == 0
		if err := tx.Create(&user).Error; err != nil {
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

	token, err := s.issueSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	return &user, token, nil
}

func (s *Service) Logout(ctx context.Context, authorization string) error {
	token := normalizeBearer(authorization)
	if token == "" {
		return nil
	}
	if err := s.orm.WithContext(ctx).
		Where("token = ?", authcrypto.HashToken(token)).
		Delete(&schemas.Session{}).Error; err != nil {
		return errors.Internal("failed to delete session", err)
	}
	return nil
}

func (s *Service) Authenticate(ctx context.Context, authorization string) (authcontext.Identity, error) {
	token := normalizeBearer(authorization)
	if token == "" {
		return authcontext.Identity{}, errors.Unauthorized("missing auth token")
	}

	var out struct {
		UserID    int64
		Email     string
		ExpiresAt time.Time
	}
	err := s.orm.WithContext(ctx).
		Table("sessions s").
		Select("u.id as user_id, u.email as email, s.expires_at as expires_at").
		Joins("join users u on u.id = s.user_id").
		Where("s.token = ?", authcrypto.HashToken(token)).
		Scan(&out).Error
	if err != nil {
		return authcontext.Identity{}, errors.Internal("failed to verify session", err)
	}
	if out.UserID == 0 {
		return authcontext.Identity{}, errors.Unauthorized("invalid auth token")
	}
	if time.Now().After(out.ExpiresAt) {
		return authcontext.Identity{}, errors.Unauthorized("expired auth token")
	}
	return authcontext.Identity{UserID: out.UserID, Email: out.Email}, nil
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

func (s *Service) issueSession(ctx context.Context, userID int64) (string, error) {
	token, err := authcrypto.NewToken()
	if err != nil {
		return "", errors.Internal("failed to generate token", err)
	}
	session := schemas.Session{
		Token:     authcrypto.HashToken(token),
		UserID:    userID,
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	if err := s.orm.WithContext(ctx).Create(&session).Error; err != nil {
		return "", errors.Internal("failed to create session", err)
	}
	return token, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeBearer(authorization string) string {
	return strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
}

func validEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
