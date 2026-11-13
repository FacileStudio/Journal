-- +goose Up
ALTER TABLE alert_rules ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE api_keys ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE log_entries
    ALTER COLUMN created_at SET DEFAULT now(),
    ALTER COLUMN received_at SET DEFAULT now();
ALTER TABLE saved_queries ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE source_maps ALTER COLUMN created_at SET DEFAULT now();

-- +goose Down
ALTER TABLE source_maps ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE saved_queries ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE log_entries
    ALTER COLUMN received_at DROP DEFAULT,
    ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE api_keys ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE alert_rules ALTER COLUMN created_at DROP DEFAULT;
