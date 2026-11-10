package auth

import (
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
func NewUserStore(sqlDB *sql.DB) *kit.Store {
	return &kit.Store{DB: sqlDB, Columns: kit.Columns{Admin: "is_admin"}}
}
