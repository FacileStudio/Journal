-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_log_entries_workspace_app_created_at ON log_entries (workspace_id, app, created_at);
ALTER TABLE log_entries SET (autovacuum_vacuum_scale_factor = 0.02, autovacuum_analyze_scale_factor = 0.01);
