package schemas

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&LogEntry{}, &User{}, &Session{}, &APIKey{}, &SavedQuery{}, &AlertRule{}); err != nil {
		return err
	}

	statements := []string{
		`ALTER TABLE log_entries ADD COLUMN IF NOT EXISTS search tsvector GENERATED ALWAYS AS (to_tsvector('simple', coalesce(message, ''))) STORED`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_search ON log_entries USING GIN(search)`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_app_created_at ON log_entries (app, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_created_at_id ON log_entries (created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_meta_request_id ON log_entries ((meta->>'request_id')) WHERE meta ? 'request_id'`,
		`ALTER TABLE alert_rules DROP CONSTRAINT IF EXISTS fk_alert_rules_saved_query`,
		`ALTER TABLE alert_rules ADD CONSTRAINT fk_alert_rules_saved_query FOREIGN KEY (saved_query_id) REFERENCES saved_queries(id) ON DELETE RESTRICT`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
