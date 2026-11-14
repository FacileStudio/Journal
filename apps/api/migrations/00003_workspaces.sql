-- +goose Up
CREATE TABLE workspaces (
    id bigserial PRIMARY KEY,
    name text NOT NULL,
    slug text UNIQUE,
    personal_of bigint UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    plan_exempt boolean NOT NULL DEFAULT false,
    created_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE TABLE workspace_members (
    workspace_id bigint NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('owner', 'member')),
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id)
);

CREATE INDEX idx_workspace_members_user ON workspace_members (user_id);

CREATE TABLE workspace_usage (
    workspace_id bigint NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    day date NOT NULL,
    count bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (workspace_id, day)
);

INSERT INTO workspaces (id, name, plan_exempt) VALUES (1, 'Facile', true);

SELECT setval(pg_get_serial_sequence('workspaces', 'id'), 1);

INSERT INTO workspace_members (workspace_id, user_id, role)
SELECT 1, id, CASE WHEN is_admin THEN 'owner' ELSE 'member' END FROM users;

CREATE TABLE workspace_purges (
    workspace_id bigint PRIMARY KEY,
    deleted_at timestamp with time zone NOT NULL DEFAULT now()
);

-- +goose StatementBegin
CREATE FUNCTION workspace_purge_logs() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO workspace_purges (workspace_id) VALUES (OLD.id) ON CONFLICT DO NOTHING;
    RETURN OLD;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER workspaces_purge_logs AFTER DELETE ON workspaces
    FOR EACH ROW EXECUTE FUNCTION workspace_purge_logs();
