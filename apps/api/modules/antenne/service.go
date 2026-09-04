package antenne

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	antenneclient "github.com/FacileStudio/antenne-client/go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	enveloppe "github.com/FacileStudio/enveloppe/go"

	stderrors "errors"
)

// Service manages the Antenne pool connection and its settings.
type Service struct {
	orm    *gorm.DB
	client *antenneclient.Client
	logger *slog.Logger
	mu     sync.RWMutex
}

// NewService constructs a Service bound to orm and logger.
func NewService(orm *gorm.DB, logger *slog.Logger) *Service {
	return &Service{orm: orm, logger: logger}
}

func getEnvPoolConfig() (string, string) {
	return os.Getenv("ANTENNE_URL"), os.Getenv("ANTENNE_SECRET")
}

// AutoConnect loads settings and attempts to connect if they are configured.
func (s *Service) AutoConnect(ctx context.Context) {
	settings, fromEnv, err := s.getSettings(ctx)
	if err != nil {
		s.logger.Error("antenne: failed to load settings", slog.Any("error", err))
		return
	}
	if settings.Enabled && settings.URL != "" && settings.Secret != "" {
		if err := s.connect(settings.URL, settings.Secret); err != nil {
			s.logger.Error("antenne: auto-connect failed", slog.Any("error", err))
		}
		return
	}

	if fromEnv && settings.URL != "" && settings.Secret != "" {
		s.logger.Info("antenne: using env vars for auto-connect")
		if err := s.connect(settings.URL, settings.Secret); err != nil {
			s.logger.Error("antenne: auto-connect from env failed", slog.Any("error", err))
		}
	}
}

func (s *Service) getSettings(ctx context.Context) (*PoolSettings, bool, error) {
	var record schemas.AppSetting
	err := s.orm.WithContext(ctx).Where("id = ?", 1).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		envURL, envSecret := getEnvPoolConfig()
		if envURL != "" && envSecret != "" {
			return &PoolSettings{URL: envURL, Secret: envSecret, Enabled: false}, true, nil
		}
		return &PoolSettings{}, false, nil
	}
	if err != nil {
		return nil, false, errors.Internal("failed to get pool settings", err)
	}
	return &PoolSettings{URL: record.AntenneURL, Secret: record.AntenneSecret, Enabled: record.AntenneEnabled}, false, nil
}

func (s *Service) updateSettings(ctx context.Context, req *UpdatePoolRequest) (*PoolSettings, string, error) {
	record := schemas.AppSetting{
		ID:             1,
		AntenneURL:     strings.TrimSpace(req.URL),
		AntenneSecret:  strings.TrimSpace(req.Secret),
		AntenneEnabled: req.Enabled,
	}
	if err := s.orm.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"antenne_url", "antenne_secret", "antenne_enabled"}),
	}).Create(&record).Error; err != nil {
		return nil, "", errors.Internal("failed to update pool settings", err)
	}

	var connectErr string
	if req.Enabled && req.URL != "" && req.Secret != "" {
		if err := s.connect(req.URL, req.Secret); err != nil {
			s.logger.Error("antenne: connect failed after settings update", slog.Any("error", err))
			connectErr = err.Error()
		}
	} else {
		s.disconnect()
	}

	return &PoolSettings{URL: record.AntenneURL, Secret: record.AntenneSecret, Enabled: record.AntenneEnabled}, connectErr, nil
}

func (s *Service) connect(instanceURL, secret string) error {
	s.disconnect()

	cfg := &antenneclient.Config{
		App:      "Journal",
		Instance: instanceURL,
		Secret:   secret,
		Events: antenneclient.EventConfig{
			Emit:   []string{"alert.fired"},
			Listen: []string{},
		},
	}

	client := antenneclient.NewClient(cfg,
		antenneclient.WithOnConnect(func() { s.logger.Info("antenne: connected") }),
		antenneclient.WithOnDisconnect(func() { s.logger.Info("antenne: disconnected") }),
		antenneclient.WithOnError(func(err error) {
			s.logger.Error("antenne: error", slog.Any("error", err))
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	s.client = client
	s.mu.Unlock()

	s.logger.Info("antenne: connected")
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

func (s *Service) isConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client != nil && s.client.IsConnected()
}

func (s *Service) Shutdown() {
	s.disconnect()
}

func (s *Service) EmitAlert(alert schemas.AlertRule, count int64) {
	if !s.shouldEmit() {
		return
	}

	channel := "alert.fired"

	evt := enveloppe.Event[map[string]any]{
		Version:  enveloppe.EventVersion,
		App:      enveloppe.App("Journal"),
		Object:   enveloppe.ObjectType("unknown"),
		Action:   enveloppe.ActionCreated,
		FacileID: fmt.Sprintf("journal_alert_%d", alert.ID),
		Payload: map[string]any{
			"alert": map[string]any{
				"id":             alert.ID,
				"name":           alert.Name,
				"threshold":      alert.Threshold,
				"window_minutes": alert.WindowMinutes,
				"saved_query_id": alert.SavedQueryID,
			},
			"count":    count,
			"fired_at": time.Now().UTC().Format(time.RFC3339),
		},
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		IdempotencyKey: fmt.Sprintf("journal_alert_%d_%d", alert.ID, time.Now().UnixMilli()),
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		s.logger.Error("antenne: failed to serialize alert event", slog.Any("error", err))
		return
	}

	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client != nil && client.IsConnected() {
		if err := client.EmitNow(channel, json.RawMessage(payload)); err != nil {
			s.logger.Error("antenne: failed to emit alert event", slog.Any("error", err))
		}
	} else {
		s.logger.Warn("antenne: not connected, dropping alert event")
	}
}

func (s *Service) shouldEmit() bool {
	s.mu.RLock()
	hasClient := s.client != nil
	s.mu.RUnlock()
	if hasClient {
		return true
	}
	var record schemas.AppSetting
	if err := s.orm.Where("id = ?", 1).First(&record).Error; err != nil {
		return false
	}
	return record.AntenneEnabled
}
