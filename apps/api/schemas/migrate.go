package schemas

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&LogEntry{}, &User{}, &Session{}); err != nil {
		return err
	}

	statements := []string{
		`ALTER TABLE log_entries ADD COLUMN IF NOT EXISTS search tsvector GENERATED ALWAYS AS (to_tsvector('simple', coalesce(message, ''))) STORED`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_search ON log_entries USING GIN(search)`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_app_created_at ON log_entries (app, created_at DESC)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
