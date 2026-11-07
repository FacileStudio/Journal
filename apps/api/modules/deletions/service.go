// Package deletions erases the people Registre erased.
package deletions

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/FacileStudio/porte/directory"
	"gorm.io/gorm"
)

// Service receives Registre's GDPR deletion notices. It polls Registre's
// deletion list with the app's own OIDC client credentials and accepts
// Registre's signed push on directory.DeletionsPath. Journal does not sync
// Registre spaces, so this is the only use it makes of the directory client.
type Service struct {
	orm      *gorm.DB
	logger   *slog.Logger
	receiver *directory.Deletions
}

// NewService builds the receiver over Journal's tables. erase removes porte's
// own rows for the identity: pg.Store.EraseIdentity.
func NewService(
	orm *gorm.DB, client directory.Client, erase func(context.Context, string, string) error, logger *slog.Logger,
) *Service {
	service := &Service{orm: orm, logger: logger}
	service.receiver = &directory.Deletions{
		Client:          client,
		OnPersonDeleted: service.ErasePerson,
		Erase:           erase,
		State:           state{orm: orm},
		Logger:          logger,
	}
	return service
}

// Run polls until ctx ends. Start it in exactly one replica.
func (s *Service) Run(ctx context.Context) {
	s.receiver.Run(ctx)
}

// Handler verifies and applies Registre's signed user.deleted push.
func (s *Service) Handler() http.Handler {
	return s.receiver.Handler()
}

// ErasePerson deletes the account behind a Registre identity. The row's
// foreign keys cascade into the identity with its provider tokens, the
// sessions (named tokens included) and the login codes. Journal keeps no other
// per-person row: ingest keys, saved queries, alert rules and source maps
// belong to the instance, not to a person. A subject that is already gone is a
// no-op, so a replayed notice changes nothing.
//
// Unlike a self-service delete, this one never refuses for the last
// administrator: an erasure Registre has already performed cannot be blocked
// here. It logs a warning when no administrator remains instead.
func (s *Service) ErasePerson(ctx context.Context, provider, subject string) error {
	var userIDs []int64
	err := s.orm.WithContext(ctx).Raw(
		`SELECT user_id FROM porte_identities WHERE provider = ? AND subject = ?`, provider, subject,
	).Scan(&userIDs).Error
	if err != nil || len(userIDs) == 0 {
		return err
	}
	if err := s.orm.WithContext(ctx).Exec(`DELETE FROM users WHERE id = ?`, userIDs[0]).Error; err != nil {
		return err
	}
	s.warnWithoutAdmin(ctx)
	return nil
}

func (s *Service) warnWithoutAdmin(ctx context.Context) {
	var orphaned bool
	err := s.orm.WithContext(ctx).Raw(
		`SELECT EXISTS (SELECT 1 FROM users) AND NOT EXISTS (SELECT 1 FROM users WHERE is_admin)`,
	).Scan(&orphaned).Error
	if err == nil && orphaned {
		s.logger.Warn("an erased person was the last administrator: promote another account in the database")
	}
}

type state struct {
	orm *gorm.DB
}

// Load returns the saved state, or nil before the first save.
func (s state) Load(ctx context.Context) ([]byte, error) {
	var rows []struct{ State []byte }
	err := s.orm.WithContext(ctx).Raw(`SELECT state FROM directory_deletion_state WHERE id = 1`).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0].State, nil
}

// Save replaces the saved state.
func (s state) Save(ctx context.Context, saved []byte) error {
	return s.orm.WithContext(ctx).Exec(
		`INSERT INTO directory_deletion_state (id, state, updated_at) VALUES (1, ?, now())
		 ON CONFLICT (id) DO UPDATE SET state = EXCLUDED.state, updated_at = now()`, saved).Error
}
