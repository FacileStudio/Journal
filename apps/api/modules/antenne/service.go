package antenne

import (
	"context"
	"database/sql"
	stderrors "errors"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	antenneclient "github.com/FacileStudio/antenne-client/go"
	"github.com/FacileStudio/antenne-client/go/catalog"
	"github.com/FacileStudio/antenne-client/go/envelope"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service manages the Antenne pool connection, its settings, and the outbox
// that makes Journal's alert events durable.
type Service struct {
	orm    *gorm.DB
	db     *sql.DB
	client *antenneclient.Client
	logger *slog.Logger
	mu     sync.RWMutex
	dialMu sync.Mutex
}

// NewService constructs a Service bound to the database and logger. db is the
// handle under orm: the outbox runs on database/sql.
func NewService(orm *gorm.DB, db *sql.DB, logger *slog.Logger) *Service {
	return &Service{orm: orm, db: db, logger: logger}
}

// poolConfig is the stored connection, secret included. It stays inside the
// package: the API shows PoolSettings, which carries no secret.
type poolConfig struct {
	URL     string
	Secret  string
	Enabled bool
}

func (c poolConfig) view() PoolSettings {
	return PoolSettings{URL: c.URL, SecretSet: c.Secret != "", Enabled: c.Enabled}
}

func (s *Service) getSettings(ctx context.Context) (*poolConfig, bool, error) {
	var record schemas.AppSetting
	err := s.orm.WithContext(ctx).Where("id = ?", 1).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return envSettings()
	}
	if err != nil {
		return nil, false, errors.Internal("failed to get pool settings", err)
	}
	return &poolConfig{URL: record.AntenneURL, Secret: record.AntenneSecret, Enabled: record.AntenneEnabled}, false, nil
}

// envSettings reads ANTENNE_URL, or its older synonym ANTENNE_INSTANCE, and
// ANTENNE_SECRET through the client library. Nothing set means no settings;
// a URL without a secret is a configuration error.
func envSettings() (*poolConfig, bool, error) {
	cfg, err := antenneclient.ConfigFromEnv(string(envelope.AppJournal))
	if stderrors.Is(err, antenneclient.ErrNotConfigured) {
		return &poolConfig{}, false, nil
	}
	if err != nil {
		return nil, false, errors.Internal("invalid Antenne environment", err)
	}
	return &poolConfig{URL: cfg.Instance, Secret: cfg.Secret}, true, nil
}

func (s *Service) updateSettings(ctx context.Context, req *UpdatePoolRequest) (*poolConfig, string, error) {
	current, _, err := s.getSettings(ctx)
	if err != nil {
		return nil, "", err
	}
	next := poolConfig{URL: strings.TrimSpace(req.URL), Secret: current.Secret, Enabled: req.Enabled}
	if req.Secret != nil {
		next.Secret = strings.TrimSpace(*req.Secret)
	}
	record := schemas.AppSetting{
		ID:             1,
		AntenneURL:     next.URL,
		AntenneSecret:  next.Secret,
		AntenneEnabled: next.Enabled,
	}
	if err := s.orm.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"antenne_url", "antenne_secret", "antenne_enabled"}),
	}).Create(&record).Error; err != nil {
		return nil, "", errors.Internal("failed to update pool settings", err)
	}

	var connectErr string
	if next.Enabled && next.URL != "" && next.Secret != "" {
		if err := s.connect(next.URL, next.Secret); err != nil {
			s.logger.Error("antenne: connect failed after settings update", slog.Any("error", err))
			connectErr = err.Error()
		}
	} else {
		s.disconnect()
	}

	return &next, connectErr, nil
}

// connect replaces the live client. The client keeps reconnecting for as long
// as the process runs: the library's default gives up after 20 attempts,
// about seven minutes, which a suite-wide redeploy outlasts.
func (s *Service) connect(instanceURL, secret string) error {
	s.dialMu.Lock()
	defer s.dialMu.Unlock()
	s.disconnect()

	cfg := &antenneclient.Config{
		App:      string(envelope.AppJournal),
		Instance: instanceURL,
		Secret:   secret,
		Events:   antenneclient.EventConfig{Emit: []string{string(catalog.AlertFired)}, Listen: []string{}},
	}
	client := antenneclient.NewClient(cfg,
		antenneclient.WithMaxReconnect(math.MaxInt32),
		antenneclient.WithOnConnect(func() { s.logger.Info("antenne: connected") }),
		antenneclient.WithOnDisconnect(func() { s.logger.Info("antenne: disconnected") }),
		antenneclient.WithOnError(func(err error) { s.logger.Error("antenne: error", slog.Any("error", err)) }),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Connect(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	s.client = client
	s.mu.Unlock()
	return nil
}

func (s *Service) disconnect() {
	s.mu.Lock()
	client := s.client
	s.client = nil
	s.mu.Unlock()

	if client != nil {
		client.Disconnect()
	}
}

func (s *Service) current() *antenneclient.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
}

func (s *Service) isConnected() bool {
	client := s.current()
	return client != nil && client.IsConnected()
}

// Shutdown closes the live connection.
func (s *Service) Shutdown() {
	s.disconnect()
}
