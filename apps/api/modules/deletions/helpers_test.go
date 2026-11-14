package deletions

import (
	"context"
	"strconv"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"gorm.io/gorm"
)

func seedPerson(t *testing.T, db *gorm.DB, issuer, email, subject string, admin bool) int64 {
	t.Helper()
	user := schemas.User{Email: email, Name: email, IsAdmin: admin}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	tag := strconv.FormatInt(user.ID, 10)
	statements := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO porte_identities (user_id, provider, subject, refresh_token) VALUES (?, ?, ?, 'refresh')`, []any{user.ID, issuer, subject}},
		{`INSERT INTO porte_sessions (token_hash, user_id, label) VALUES (?, ?, 'cli')`, []any{"hash-" + tag, user.ID}},
		{`INSERT INTO porte_login_codes (code_hash, user_id, expires_at) VALUES (?, ?, now() + interval '1 hour')`, []any{"code-" + tag, user.ID}},
	}
	for _, statement := range statements {
		if err := db.Exec(statement.sql, statement.args...).Error; err != nil {
			t.Fatalf("seed credentials: %v", err)
		}
	}
	return user.ID
}

// seedPersonalWorkspace gives gone a personal workspace holding a key and a
// log line, and makes other a member of the instance workspace.
func seedPersonalWorkspace(t *testing.T, db *gorm.DB, gone, other int64) int64 {
	t.Helper()
	memberships, err := workspace.NewService(db).Memberships(context.Background(), gone, "Gone")
	if err != nil || len(memberships) != 1 {
		t.Fatalf("personal workspace: %v %+v", err, memberships)
	}
	id := memberships[0].ID
	if err := db.Create(&schemas.WorkspaceMember{WorkspaceID: 1, UserID: other, Role: schemas.RoleOwner}).Error; err != nil {
		t.Fatalf("seed member: %v", err)
	}
	if err := db.Create(&schemas.APIKey{WorkspaceID: id, App: "mine", Kind: schemas.KeyKindSecret, Prefix: "q", KeyHash: "g"}).Error; err != nil {
		t.Fatalf("seed personal key: %v", err)
	}
	if err := db.Create(&schemas.LogEntry{WorkspaceID: id, App: "mine", Level: "info", Message: "personal"}).Error; err != nil {
		t.Fatalf("seed personal log: %v", err)
	}
	return id
}
