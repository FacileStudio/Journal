-- +goose Up
CREATE TABLE alert_rules (
    id bigint NOT NULL,
    name text NOT NULL,
    saved_query_id bigint NOT NULL,
    provider text DEFAULT 'webhook'::text NOT NULL,
    threshold bigint NOT NULL,
    window_minutes bigint NOT NULL,
    webhook_url text NOT NULL,
    webhook_header text,
    webhook_secret text,
    enabled boolean DEFAULT true NOT NULL,
    last_fired_at timestamp with time zone,
    created_at timestamp with time zone
);

CREATE SEQUENCE alert_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE alert_rules_id_seq OWNED BY alert_rules.id;

CREATE TABLE antenne_outbox (
    id bigint NOT NULL,
    channel text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    idempotency_key text NOT NULL,
    sent_at timestamp with time zone
);

CREATE SEQUENCE antenne_outbox_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE antenne_outbox_id_seq OWNED BY antenne_outbox.id;

CREATE TABLE api_key_usage (
    api_key_id bigint NOT NULL,
    day date NOT NULL,
    count bigint DEFAULT 0 NOT NULL
);

CREATE TABLE api_keys (
    id bigint NOT NULL,
    app text NOT NULL,
    kind text DEFAULT 'secret'::text NOT NULL,
    prefix text NOT NULL,
    key_hash text NOT NULL,
    allowed_origins jsonb,
    daily_quota bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone,
    revoked_at timestamp with time zone
);

CREATE SEQUENCE api_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE api_keys_id_seq OWNED BY api_keys.id;

CREATE TABLE app_settings (
    id bigint NOT NULL,
    antenne_url text DEFAULT ''::text NOT NULL,
    antenne_secret text DEFAULT ''::text NOT NULL,
    antenne_enabled boolean DEFAULT false NOT NULL
);

CREATE SEQUENCE app_settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE app_settings_id_seq OWNED BY app_settings.id;

CREATE TABLE directory_deletion_state (
    id integer NOT NULL,
    state bytea NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE log_entries (
    id bigint NOT NULL,
    app text NOT NULL,
    level text DEFAULT 'info'::text,
    message text,
    meta jsonb,
    created_at timestamp with time zone,
    received_at timestamp with time zone,
    search tsvector GENERATED ALWAYS AS (to_tsvector('simple'::regconfig, COALESCE(message, ''::text))) STORED
);

CREATE SEQUENCE log_entries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE log_entries_id_seq OWNED BY log_entries.id;

CREATE TABLE mcp_actions (
    id text NOT NULL,
    user_id text NOT NULL,
    kind text NOT NULL,
    target text NOT NULL,
    digest text NOT NULL,
    summary text DEFAULT ''::text NOT NULL,
    client_id text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    resolved_at timestamp with time zone,
    outcome text DEFAULT ''::text NOT NULL
);

CREATE TABLE porte_identities (
    user_id bigint NOT NULL,
    provider text NOT NULL,
    subject text NOT NULL,
    password_hash text DEFAULT ''::text NOT NULL,
    access_token text DEFAULT ''::text NOT NULL,
    refresh_token text DEFAULT ''::text NOT NULL,
    token_expiry timestamp with time zone,
    roles jsonb,
    roles_synced_at timestamp with time zone,
    synced_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now()
);

CREATE TABLE porte_login_codes (
    code_hash text NOT NULL,
    user_id bigint NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    consumed_at timestamp with time zone,
    provider text DEFAULT ''::text NOT NULL,
    subject text DEFAULT ''::text NOT NULL
);

CREATE TABLE porte_sessions (
    id bigint NOT NULL,
    token_hash text NOT NULL,
    user_id bigint NOT NULL,
    label text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_used_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone
);

CREATE SEQUENCE porte_sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE porte_sessions_id_seq OWNED BY porte_sessions.id;

CREATE TABLE saved_queries (
    id bigint NOT NULL,
    name text NOT NULL,
    params jsonb NOT NULL,
    created_at timestamp with time zone
);

CREATE SEQUENCE saved_queries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE saved_queries_id_seq OWNED BY saved_queries.id;

CREATE TABLE source_maps (
    id bigint NOT NULL,
    app text NOT NULL,
    release text NOT NULL,
    file text NOT NULL,
    content text NOT NULL,
    bytes bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone
);

CREATE SEQUENCE source_maps_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE source_maps_id_seq OWNED BY source_maps.id;

CREATE TABLE users (
    id bigint NOT NULL,
    email text NOT NULL,
    name text,
    password_hash text DEFAULT ''::text NOT NULL,
    is_admin boolean DEFAULT false,
    avatar_url text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    locale text DEFAULT 'en'::text NOT NULL
);

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE users_id_seq OWNED BY users.id;

ALTER TABLE ONLY alert_rules ALTER COLUMN id SET DEFAULT nextval('alert_rules_id_seq'::regclass);

ALTER TABLE ONLY antenne_outbox ALTER COLUMN id SET DEFAULT nextval('antenne_outbox_id_seq'::regclass);

ALTER TABLE ONLY api_keys ALTER COLUMN id SET DEFAULT nextval('api_keys_id_seq'::regclass);

ALTER TABLE ONLY app_settings ALTER COLUMN id SET DEFAULT nextval('app_settings_id_seq'::regclass);

ALTER TABLE ONLY log_entries ALTER COLUMN id SET DEFAULT nextval('log_entries_id_seq'::regclass);

ALTER TABLE ONLY porte_sessions ALTER COLUMN id SET DEFAULT nextval('porte_sessions_id_seq'::regclass);

ALTER TABLE ONLY saved_queries ALTER COLUMN id SET DEFAULT nextval('saved_queries_id_seq'::regclass);

ALTER TABLE ONLY source_maps ALTER COLUMN id SET DEFAULT nextval('source_maps_id_seq'::regclass);

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);

ALTER TABLE ONLY alert_rules
    ADD CONSTRAINT alert_rules_pkey PRIMARY KEY (id);

ALTER TABLE ONLY antenne_outbox
    ADD CONSTRAINT antenne_outbox_idempotency_key_key UNIQUE (idempotency_key);

ALTER TABLE ONLY antenne_outbox
    ADD CONSTRAINT antenne_outbox_pkey PRIMARY KEY (id);

ALTER TABLE ONLY api_key_usage
    ADD CONSTRAINT api_key_usage_pkey PRIMARY KEY (api_key_id, day);

ALTER TABLE ONLY api_keys
    ADD CONSTRAINT api_keys_pkey PRIMARY KEY (id);

ALTER TABLE ONLY app_settings
    ADD CONSTRAINT app_settings_pkey PRIMARY KEY (id);

ALTER TABLE ONLY directory_deletion_state
    ADD CONSTRAINT directory_deletion_state_pkey PRIMARY KEY (id);

ALTER TABLE ONLY log_entries
    ADD CONSTRAINT log_entries_pkey PRIMARY KEY (id);

ALTER TABLE ONLY mcp_actions
    ADD CONSTRAINT mcp_actions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY porte_identities
    ADD CONSTRAINT porte_identities_pkey PRIMARY KEY (provider, subject);

ALTER TABLE ONLY porte_login_codes
    ADD CONSTRAINT porte_login_codes_pkey PRIMARY KEY (code_hash);

ALTER TABLE ONLY porte_sessions
    ADD CONSTRAINT porte_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY porte_sessions
    ADD CONSTRAINT porte_sessions_token_hash_key UNIQUE (token_hash);

ALTER TABLE ONLY saved_queries
    ADD CONSTRAINT saved_queries_pkey PRIMARY KEY (id);

ALTER TABLE ONLY source_maps
    ADD CONSTRAINT source_maps_pkey PRIMARY KEY (id);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

CREATE INDEX antenne_outbox_sent ON antenne_outbox USING btree (sent_at) WHERE (sent_at IS NOT NULL);

CREATE INDEX antenne_outbox_unsent ON antenne_outbox USING btree (id) WHERE (sent_at IS NULL);

CREATE INDEX idx_alert_rules_provider ON alert_rules USING btree (provider) WHERE enabled;

CREATE UNIQUE INDEX idx_api_keys_key_hash ON api_keys USING btree (key_hash);

CREATE INDEX idx_api_keys_kind ON api_keys USING btree (kind) WHERE (revoked_at IS NULL);

CREATE INDEX idx_log_entries_app ON log_entries USING btree (app);

CREATE INDEX idx_log_entries_app_created_at ON log_entries USING btree (app, created_at DESC);

CREATE INDEX idx_log_entries_created_at ON log_entries USING btree (created_at);

CREATE INDEX idx_log_entries_created_at_id ON log_entries USING btree (created_at DESC, id DESC);

CREATE INDEX idx_log_entries_level ON log_entries USING btree (level);

CREATE INDEX idx_log_entries_meta_request_id ON log_entries USING btree (((meta ->> 'request_id'::text))) WHERE (meta ? 'request_id'::text);

CREATE INDEX idx_log_entries_meta_session_id ON log_entries USING btree (((meta ->> 'session_id'::text))) WHERE (meta ? 'session_id'::text);

CREATE INDEX idx_log_entries_meta_source ON log_entries USING btree (((meta ->> 'source'::text)), created_at DESC) WHERE (meta ? 'source'::text);

CREATE INDEX idx_log_entries_search ON log_entries USING gin (search);

CREATE UNIQUE INDEX idx_saved_queries_name ON saved_queries USING btree (name);

CREATE UNIQUE INDEX idx_source_maps_lookup ON source_maps USING btree (app, release, file);

CREATE UNIQUE INDEX idx_users_email ON users USING btree (email);

CREATE INDEX mcp_actions_open_idx ON mcp_actions USING btree (user_id, created_at DESC) WHERE (resolved_at IS NULL);

CREATE INDEX porte_identities_user_idx ON porte_identities USING btree (user_id);

CREATE INDEX porte_sessions_expiry_idx ON porte_sessions USING btree (expires_at) WHERE (expires_at IS NOT NULL);

CREATE INDEX porte_sessions_user_idx ON porte_sessions USING btree (user_id);

ALTER TABLE ONLY alert_rules
    ADD CONSTRAINT fk_alert_rules_saved_query FOREIGN KEY (saved_query_id) REFERENCES saved_queries(id) ON DELETE RESTRICT;

ALTER TABLE ONLY api_key_usage
    ADD CONSTRAINT fk_api_key_usage_key FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE;

ALTER TABLE ONLY porte_identities
    ADD CONSTRAINT porte_identities_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY porte_login_codes
    ADD CONSTRAINT porte_login_codes_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY porte_sessions
    ADD CONSTRAINT porte_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
