package schemas_test

import (
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	troncdb "github.com/FacileStudio/tronc/testdb"
)

var models = []any{
	&schemas.LogEntry{}, &schemas.User{}, &schemas.APIKey{}, &schemas.APIKeyUsage{},
	&schemas.SavedQuery{}, &schemas.AlertRule{}, &schemas.SourceMap{}, &schemas.AppSetting{},
	&schemas.Workspace{}, &schemas.WorkspaceMember{},
}

// TestGORMModelsMatchTheMigratedSchema checks every column a GORM model reads
// or writes exists in the schema the migrations build, with a type the field
// can hold. AutoMigrate no longer adds a missing column, so a field added to a
// struct without a migration fails here instead of at query time.
func TestGORMModelsMatchTheMigratedSchema(t *testing.T) {
	troncdb.AssertModelsMatch(t, testdb.Migrated(t), models...)
}
