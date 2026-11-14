-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_log_entries_workspace_created_at_id ON log_entries (workspace_id, created_at DESC, id DESC);
