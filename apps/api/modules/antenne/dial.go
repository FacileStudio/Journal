package antenne

import (
	"context"
	"log/slog"
	"time"
)

// AutoConnect dials the configured instance in the background and retries
// with backoff, capped at a minute, until it connects, the settings turn
// publishing off, or ctx ends. The client library does not retry a first dial
// that fails, so without this an Antenne that is down at Journal's boot would
// end publishing until the next restart.
func (s *Service) AutoConnect(ctx context.Context) {
	go s.dialUntilConnected(ctx)
}

func (s *Service) dialUntilConnected(ctx context.Context) {
	wait := time.Second
	for !s.dialOnce(ctx) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = min(wait*2, time.Minute)
	}
}

// dialOnce reports whether nothing is left to dial: a client is live, or the
// settings ask for none.
func (s *Service) dialOnce(ctx context.Context) bool {
	if s.current() != nil {
		return true
	}
	settings, fromEnv, err := s.getSettings(ctx)
	if err != nil {
		s.logger.Error("antenne: failed to load settings", slog.Any("error", err))
		return ctx.Err() != nil
	}
	if !wanted(settings, fromEnv) {
		return true
	}
	if err := s.connect(settings.URL, settings.Secret); err != nil {
		s.logger.Error("antenne: auto-connect failed", slog.Any("error", err))
		return false
	}
	return true
}

// wanted reports whether the settings ask for a connection: enabled in the
// dashboard, or given by the environment, with both a URL and a secret.
func wanted(settings *PoolSettings, fromEnv bool) bool {
	return (settings.Enabled || fromEnv) && settings.URL != "" && settings.Secret != ""
}
