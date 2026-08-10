package apikeys

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	stderrors "errors"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/authcrypto"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

const (
	tokenNamespace   = "journal_"
	publicMarker     = "pub_"
	prefixRandomLen  = 6
	tokenRandomBytes = 32
	maxAllowedOrigin = 8
	maxDailyQuota    = 10_000_000
)

var appNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// NewKey is what an admin asks for. Kind is empty or schemas.KeyKindSecret for
// a server credential; the origins and the quota are only read for a public
// one, where both are mandatory.
type NewKey struct {
	App            string
	Kind           string
	AllowedOrigins []string
	DailyQuota     int
}

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

// UsageToday reports how much of each key's daily quota is already spent, so
// the dashboard can show a public key running out before the errors stop
// arriving rather than after.
func (s *Service) UsageToday(ctx context.Context) (map[int64]int64, error) {
	var rows []schemas.APIKeyUsage
	err := s.orm.WithContext(ctx).
		Where("day = ?", time.Now().UTC().Format(time.DateOnly)).
		Find(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to read api key usage", err)
	}
	usage := make(map[int64]int64, len(rows))
	for _, row := range rows {
		usage[row.APIKeyID] = row.Count
	}
	return usage, nil
}

func (s *Service) Create(ctx context.Context, request NewKey) (*schemas.APIKey, string, error) {
	if !validAppName(request.App) {
		return nil, "", errors.Invalid("app must match ^[a-z0-9][a-z0-9-]{0,63}$")
	}

	kind := request.Kind
	if kind == "" {
		kind = schemas.KeyKindSecret
	}
	if kind != schemas.KeyKindSecret && kind != schemas.KeyKindPublic {
		return nil, "", errors.Invalid("kind must be secret or public")
	}

	key := schemas.APIKey{App: request.App, Kind: kind}
	if kind == schemas.KeyKindPublic {
		origins, err := NormalizeOrigins(request.AllowedOrigins)
		if err != nil {
			return nil, "", err
		}
		if request.DailyQuota < 1 || request.DailyQuota > maxDailyQuota {
			return nil, "", errors.Invalid("daily_quota must be between 1 and 10000000 for a public key")
		}
		key.AllowedOrigins = origins
		key.DailyQuota = request.DailyQuota
	}

	token, prefix, hash, err := generateToken(request.App, kind)
	if err != nil {
		return nil, "", errors.Internal("failed to generate api key", err)
	}
	key.Prefix = prefix
	key.KeyHash = hash

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

// VerifyIngestKey authenticates the server endpoint, and deliberately refuses
// a public key: a token that ships inside a JavaScript bundle must not reach
// the unquotaed, thousand-entry batch route.
func (s *Service) VerifyIngestKey(ctx context.Context, token string) (string, error) {
	key, err := s.lookup(ctx, token, schemas.KeyKindSecret)
	if err != nil {
		return "", err
	}
	return key.App, nil
}

// VerifyBrowserKey authenticates the browser endpoint and hands back the whole
// record, because the caller still has to check the origin and the quota.
func (s *Service) VerifyBrowserKey(ctx context.Context, token string) (schemas.APIKey, error) {
	return s.lookup(ctx, token, schemas.KeyKindPublic)
}

func (s *Service) lookup(ctx context.Context, token, kind string) (schemas.APIKey, error) {
	var key schemas.APIKey
	err := s.orm.WithContext(ctx).
		Where("key_hash = ? AND kind = ? AND revoked_at IS NULL", authcrypto.HashToken(token), kind).
		First(&key).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return schemas.APIKey{}, errors.Unauthorized("invalid ingest token")
		}
		return schemas.APIKey{}, errors.Internal("failed to verify ingest token", err)
	}
	return key, nil
}

// NormalizeOrigins turns what an admin typed into the exact strings a browser
// puts in the Origin header: scheme and host lowercased, the default port
// dropped, no path and no trailing slash.
//
// It is strict on purpose. An allowlist entry that never matches is an outage
// that looks like a bug in the SDK, and one that matches more than it should
// is the reason the allowlist exists.
func NormalizeOrigins(origins []string) ([]string, error) {
	if len(origins) == 0 {
		return nil, errors.Invalid("a public key needs at least one allowed origin")
	}
	if len(origins) > maxAllowedOrigin {
		return nil, errors.Invalid("a public key accepts at most 8 allowed origins")
	}

	normalized := make([]string, 0, len(origins))
	for _, raw := range origins {
		origin, err := NormalizeOrigin(raw)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(normalized, origin) {
			normalized = append(normalized, origin)
		}
	}
	return normalized, nil
}

func NormalizeOrigin(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.Invalid("an allowed origin cannot be empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.Invalid("allowed origin " + trimmed + " is not a URL")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.Invalid("allowed origin " + trimmed + " must start with http:// or https://")
	}
	if parsed.Hostname() == "" {
		return "", errors.Invalid("allowed origin " + trimmed + " has no host")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", errors.Invalid("allowed origin " + trimmed + " must carry no path")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return "", errors.Invalid("allowed origin " + trimmed + " must be scheme://host[:port] only")
	}
	if strings.Contains(parsed.Hostname(), "*") {
		return "", errors.Invalid("wildcards are not accepted in an allowed origin; list each host")
	}

	origin := scheme + "://" + strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if port != "" && !isDefaultPort(scheme, port) {
		origin += ":" + port
	}
	return origin, nil
}

func isDefaultPort(scheme, port string) bool {
	return (scheme == "http" && port == "80") || (scheme == "https" && port == "443")
}

func validAppName(app string) bool {
	return appNamePattern.MatchString(app)
}

// generateToken makes the kind readable at a glance. journal_pub_shop_… in a
// git diff or a bundle is recognisably not a credential anyone has to rotate
// in a panic; journal_shop_… is.
func generateToken(app, kind string) (string, string, string, error) {
	random := make([]byte, tokenRandomBytes)
	if _, err := rand.Read(random); err != nil {
		return "", "", "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(random)
	head := tokenNamespace
	if kind == schemas.KeyKindPublic {
		head += publicMarker
	}
	head += app + "_"
	return head + encoded, head + encoded[:prefixRandomLen], authcrypto.HashToken(head + encoded), nil
}
