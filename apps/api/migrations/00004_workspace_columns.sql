-- +goose Up
SET LOCAL lock_timeout = '10s';

ALTER TABLE log_entries ADD COLUMN workspace_id bigint NOT NULL DEFAULT 1;
ALTER TABLE log_entries ALTER COLUMN workspace_id DROP DEFAULT;
ALTER TABLE log_entries ADD CONSTRAINT log_entries_workspace_id_positive CHECK (workspace_id > 0) NOT VALID;

ALTER TABLE api_keys ADD COLUMN workspace_id bigint NOT NULL DEFAULT 1 REFERENCES workspaces (id) ON DELETE CASCADE;
ALTER TABLE api_keys ALTER COLUMN workspace_id DROP DEFAULT;
CREATE INDEX idx_api_keys_workspace ON api_keys (workspace_id);

ALTER TABLE saved_queries ADD COLUMN workspace_id bigint NOT NULL DEFAULT 1 REFERENCES workspaces (id) ON DELETE CASCADE;
ALTER TABLE saved_queries ALTER COLUMN workspace_id DROP DEFAULT;
DROP INDEX idx_saved_queries_name;
CREATE UNIQUE INDEX idx_saved_queries_workspace_name ON saved_queries (workspace_id, name);

ALTER TABLE alert_rules ADD COLUMN workspace_id bigint NOT NULL DEFAULT 1 REFERENCES workspaces (id) ON DELETE CASCADE;
ALTER TABLE alert_rules ALTER COLUMN workspace_id DROP DEFAULT;
CREATE INDEX idx_alert_rules_workspace ON alert_rules (workspace_id);

ALTER TABLE source_maps ADD COLUMN workspace_id bigint NOT NULL DEFAULT 1 REFERENCES workspaces (id) ON DELETE CASCADE;
ALTER TABLE source_maps ALTER COLUMN workspace_id DROP DEFAULT;
DROP INDEX idx_source_maps_lookup;
CREATE UNIQUE INDEX idx_source_maps_workspace_lookup ON source_maps (workspace_id, app, release, file);
