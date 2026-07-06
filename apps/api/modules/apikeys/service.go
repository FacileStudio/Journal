package apikeys

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	stderrors "errors"
	"regexp"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/authcrypto"
	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/schemas"

	"gorm.io/gorm"
)

const (
	tokenNamespace   = "journal_"
	prefixRandomLen  = 6
	tokenRandomBytes = 32
)

var appNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

func (s *Service) List(ctx context.Context) ([]schemas.APIKey, error) {
	var keys []schemas.APIKey
	if err := s.orm.WithContext(ctx).Order("created_at desc, id desc").Find(&keys).Error; err != nil {
		return nil, errors.Internal("failed to list api keys", err)
	}
	return keys, nil
}

func (s *Service) Create(ctx context.Context, app string) (*schemas.APIKey, string, error) {
	if !validAppName(app) {
		return nil, "", errors.Invalid("app must match ^[a-z0-9][a-z0-9-]{0,63}$")
	}

	token, prefix, hash, err := generateToken(app)
	if err != nil {
		return nil, "", errors.Internal("failed to generate api key", err)
	}

	key := schemas.APIKey{App: app, Prefix: prefix, KeyHash: hash}
	if err := s.orm.WithContext(ctx).Create(&key).Error; err != nil {
		return nil, "", errors.Internal("failed to store api key", err)
	}
	return &key, token, nil
}

func (s *Service) Revoke(ctx context.Context, id int64) error {
	var key schemas.APIKey
	if err := s.orm.WithContext(ctx).First(&key, id).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("api key not found")
		}
		return errors.Internal("failed to load api key", err)
	}
	if key.RevokedAt != nil {
		return nil
	}

	now := time.Now().UTC()
	if err := s.orm.WithContext(ctx).Model(&key).Update("revoked_at", now).Error; err != nil {
		return errors.Internal("failed to revoke api key", err)
	}
	return nil
}

func (s *Service) VerifyIngestKey(ctx context.Context, token string) (string, error) {
	var key schemas.APIKey
	err := s.orm.WithContext(ctx).
		Where("key_hash = ? AND revoked_at IS NULL", authcrypto.HashToken(token)).
		First(&key).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.Unauthorized("invalid ingest token")
		}
		return "", errors.Internal("failed to verify ingest token", err)
	}
	return key.App, nil
}

func validAppName(app string) bool {
	return appNamePattern.MatchString(app)
}

func generateToken(app string) (string, string, string, error) {
	random := make([]byte, tokenRandomBytes)
	if _, err := rand.Read(random); err != nil {
		return "", "", "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(random)
	token := tokenNamespace + app + "_" + encoded
	prefix := tokenNamespace + app + "_" + encoded[:prefixRandomLen]
	return token, prefix, authcrypto.HashToken(token), nil
}
