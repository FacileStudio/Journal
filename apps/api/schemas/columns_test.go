package schemas_test

import (
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/testdb"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var models = []any{
	&schemas.LogEntry{}, &schemas.User{}, &schemas.APIKey{}, &schemas.APIKeyUsage{},
	&schemas.SavedQuery{}, &schemas.AlertRule{}, &schemas.SourceMap{}, &schemas.AppSetting{},
}

var timeType = reflect.TypeFor[time.Time]()

func compatibleTypes(field *schema.Field) []string {
	switch strings.ToLower(field.TagSettings["TYPE"]) {
	case "jsonb":
		return []string{"jsonb"}
	case "text":
		return []string{"text"}
	case "date":
		return []string{"date"}
	}
	kind := field.FieldType
	for kind.Kind() == reflect.Pointer {
		kind = kind.Elem()
	}
	if kind == timeType {
		return []string{"timestamp with time zone"}
	}
	switch kind.Kind() {
	case reflect.String:
		return []string{"text"}
	case reflect.Bool:
		return []string{"boolean"}
	case reflect.Int, reflect.Int64:
		return []string{"bigint"}
	default:
		return nil
	}
}

func columnTypes(t *testing.T, db *gorm.DB, table string) map[string]string {
	t.Helper()
	var rows []struct {
		ColumnName string
		DataType   string
	}
	err := db.Raw(`SELECT column_name, data_type FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = ?`, table).Scan(&rows).Error
	if err != nil {
		t.Fatalf("read the columns of %s: %v", table, err)
	}
	columns := map[string]string{}
	for _, row := range rows {
		columns[row.ColumnName] = row.DataType
	}
	return columns
}

// TestGORMModelsMatchTheMigratedSchema checks every column a GORM model reads
// or writes exists in the schema the migrations build, with a type the field
// can hold. AutoMigrate no longer adds a missing column, so a field added to a
// struct without a migration fails here instead of at query time.
func TestGORMModelsMatchTheMigratedSchema(t *testing.T) {
	db := testdb.Migrated(t)
	for _, model := range models {
		parsed, err := schema.Parse(model, &sync.Map{}, db.NamingStrategy)
		if err != nil {
			t.Fatalf("parse %T: %v", model, err)
		}
		columns := columnTypes(t, db, parsed.Table)
		if len(columns) == 0 {
			t.Errorf("%T: table %s does not exist", model, parsed.Table)
			continue
		}
		for _, field := range parsed.Fields {
			if field.DBName == "" {
				continue
			}
			got, ok := columns[field.DBName]
			if !ok {
				t.Errorf("%s.%s: no such column", parsed.Table, field.DBName)
				continue
			}
			if want := compatibleTypes(field); !slices.Contains(want, got) {
				t.Errorf("%s.%s: column is %s, field %s needs one of %v", parsed.Table, field.DBName, got, field.FieldType, want)
			}
		}
	}
}
