package antenne

import (
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/FacileStudio/Journal/apps/api/schemas"

	antenneclient "github.com/FacileStudio/antenne-client/go"
	"github.com/FacileStudio/antenne-client/go/catalog"
	"github.com/FacileStudio/antenne-client/go/envelope"
	"github.com/FacileStudio/antenne-client/go/outbox"
)

// FireAlert records that rule fired at firedAt and, when publishing is on,
// queues its alert.fired event in the same transaction, so the rule is marked
// fired exactly when its event is durable. With publishing off the rule is
// still marked fired, as it always was, and nothing is queued into a table
// nobody drains.
func (s *Service) FireAlert(ctx context.Context, rule schemas.AlertRule, count int64, firedAt time.Time) error {
	publish := s.publishing(ctx)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE alert_rules SET last_fired_at = $1 WHERE id = $2`, firedAt, rule.ID); err != nil {
		return err
	}
	if !publish {
		s.logger.Warn("antenne: publishing is off, the alert event was not queued", slog.String("alert", rule.Name))
		return tx.Commit()
	}
	event := alertEvent(rule, count, firedAt)
	err = outbox.Enqueue(ctx, tx, catalog.AlertFired, event, event.IdempotencyKey)
	if err != nil && !stderrors.Is(err, outbox.ErrDuplicate) {
		return err
	}
	return tx.Commit()
}

// publishing reports whether an alert event should be queued: a client is
// live, or the settings ask for one.
func (s *Service) publishing(ctx context.Context) bool {
	if s.current() != nil {
		return true
	}
	settings, fromEnv, err := s.getSettings(ctx)
	return err == nil && wanted(settings, fromEnv)
}

// alertEvent is the envelope Journal has always put on alert.fired. The
// idempotency key names one firing of one rule, so a retried transaction
// cannot queue it twice.
func alertEvent(rule schemas.AlertRule, count int64, firedAt time.Time) envelope.Event[map[string]any] {
	stamp := firedAt.UTC().Format(time.RFC3339)
	return envelope.Event[map[string]any]{
		Version:  envelope.EventVersion,
		App:      envelope.AppJournal,
		Object:   envelope.ObjectType("unknown"),
		Action:   envelope.ActionCreated,
		FacileID: fmt.Sprintf("journal_alert_%d", rule.ID),
		Payload: map[string]any{
			"alert": map[string]any{
				"id":             rule.ID,
				"name":           rule.Name,
				"threshold":      rule.Threshold,
				"window_minutes": rule.WindowMinutes,
				"saved_query_id": rule.SavedQueryID,
			},
			"count":    count,
			"fired_at": stamp,
		},
		Timestamp:      stamp,
		IdempotencyKey: fmt.Sprintf("journal_alert_%d_%d", rule.ID, firedAt.UnixMilli()),
	}
}

// Drain sends queued events through the live client, in id order, until ctx
// ends. A pass stops at the first refused event and leaves it queued.
func (s *Service) Drain(ctx context.Context) {
	outbox.NewDrainer(s.db, s.emit, outbox.WithMaxBackoff(5*time.Second), outbox.WithOnError(s.drainFailed)).Run(ctx)
}

func (s *Service) emit(channel string, payload any) error {
	client := s.current()
	if client == nil {
		return antenneclient.ErrNotConnected
	}
	return client.EmitNow(channel, payload)
}

// drainFailed logs a failed pass, except the one every pass reports while the
// bus is down.
func (s *Service) drainFailed(err error) {
	if stderrors.Is(err, antenneclient.ErrNotConnected) {
		return
	}
	s.logger.Warn("antenne: outbox drain failed", slog.Any("error", err))
}
