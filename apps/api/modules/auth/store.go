package auth

import (
	"context"
	"database/sql"

	"github.com/FacileStudio/porte/kit"
)

// NewUserStore is porte/kit's store over Journal's own users table.
//
// Journal keeps that table because it carries is_admin, and the first account
// created becomes the administrator; the kit applies that rule when Columns
// names the admin flag. The kit also owns the linking rules Journal wrote by
// hand: an unverified address never reaches an existing account, an account
// that already holds a subject of the issuer is not handed to a second one,
// and the stored address is rewritten only when the provider verified it.
// Journal keeps no per-person row beyond users, whose foreign keys cascade
// into porte's identities, sessions and login codes, so there is no EraseData.
//
// kit.NewStore reads the table once: every column the kit does not write has
// a database default (schemas.Migrate gives password_hash and created_at
// theirs), so Unwritten is empty and no OnCreate is needed. It fails when the
// table or a named column is missing, which is a boot error.
func NewUserStore(ctx context.Context, sqlDB *sql.DB) (*kit.Store, error) {
	return kit.NewStore(ctx, kit.Store{DB: sqlDB, Columns: kit.Columns{Admin: "is_admin"}})
}
