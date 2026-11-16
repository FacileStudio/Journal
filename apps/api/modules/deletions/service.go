// Package deletions erases the people Registre erased.
package deletions

import (
	"context"
	"gorm.io/gorm"
	"log/slog"
)

// Service erases the data of a person Registre erased. The receiver that
// calls it is built by porte/kit.NewDeletions with ErasePerson as its callback.
type Service struct {
	orm          *gorm.DB
	logger       *slog.Logger
	eraseAccount func(context.Context, string, string) error
}

// NewService builds the erasure over Journal's tables. eraseAccount deletes
// the account behind an identity, which is porte/kit's Store.EraseAccount.
func NewService(orm *gorm.DB, eraseAccount func(context.Context, string, string) error, logger *slog.Logger) *Service {
	return &Service{orm: orm, logger: logger, eraseAccount: eraseAccount}
}

// ErasePerson deletes the account behind a Registre identity through the kit,
// in one transaction. The row's foreign keys cascade into the identity with
// its provider tokens, the sessions (named tokens included) and the login
// codes. Journal keeps no other per-person row: ingest keys, saved queries,
// alert rules and source maps belong to the instance, not to a person. A
// subject that is already gone is a no-op, so a replayed notice changes
// nothing.
//
// Unlike a self-service delete, this one never refuses for the last
// administrator: an erasure Registre has already performed cannot be blocked
// here. It logs a warning when no administrator remains instead.
func (s *Service) ErasePerson(ctx context.Context, provider, subject string) error {
	if err := s.eraseAccount(ctx, provider, subject); err != nil {
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
