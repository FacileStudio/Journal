package schemas

import (
	mcpkit "github.com/FacileStudio/tronc/mcp"
	"gorm.io/gorm"
)

// Migrate brings the schema up to date on every boot, including the replica
// that comes up second, so every statement has to be a no-op the second time.
// It runs AutoMigrate for the GORM-owned tables, then the hand-written
// statements that GORM cannot express. The meta_source index is partial,
// like request_id: only browser entries carry a source, so the index stays
// small and a filtered explorer view is a lookup rather than a scan of
// everything the suite has ever logged. And every key that predates the
// browser endpoint is a server credential — saying so in kind matters because
// an empty kind would fall through the secret check and the public check
// alike, and a key that authenticates nothing is a silent outage on the next
// deploy. password_hash and created_at get database defaults because
// porte/kit inserts an account with the columns it knows and nothing else.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&LogEntry{}, &User{}, &APIKey{}, &APIKeyUsage{}, &SavedQuery{}, &AlertRule{}, &SourceMap{}, &AppSetting{}); err != nil {
		return err
	}

	statements := []string{
		porteSchema,
		carryLegacySessionsOver,
		adoptExistingPasswords,
		`UPDATE users SET avatar_url = '' WHERE avatar_url IS NULL`,
		`ALTER TABLE users ALTER COLUMN avatar_url SET DEFAULT ''`,
		`ALTER TABLE users ALTER COLUMN avatar_url SET NOT NULL`,
		dropLocalAvatarCopies,
		`ALTER TABLE users ALTER COLUMN password_hash SET DEFAULT ''`,
		`UPDATE users SET created_at = now() WHERE created_at IS NULL`,
		`ALTER TABLE users ALTER COLUMN created_at SET DEFAULT now()`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS locale text NOT NULL DEFAULT 'en'`,
		`ALTER TABLE porte_login_codes ADD COLUMN IF NOT EXISTS provider text NOT NULL DEFAULT '', ADD COLUMN IF NOT EXISTS subject text NOT NULL DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS directory_deletion_state (id int PRIMARY KEY, state bytea NOT NULL, updated_at timestamptz NOT NULL DEFAULT now())`,
		`ALTER TABLE log_entries ADD COLUMN IF NOT EXISTS search tsvector GENERATED ALWAYS AS (to_tsvector('simple', coalesce(message, ''))) STORED`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_search ON log_entries USING GIN(search)`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_app_created_at ON log_entries (app, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_created_at_id ON log_entries (created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_meta_request_id ON log_entries ((meta->>'request_id')) WHERE meta ? 'request_id'`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_meta_session_id ON log_entries ((meta->>'session_id')) WHERE meta ? 'session_id'`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_meta_source ON log_entries ((meta->>'source'), created_at DESC) WHERE meta ? 'source'`,
		`UPDATE api_keys SET kind = 'secret' WHERE kind IS NULL OR kind = ''`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_kind ON api_keys (kind) WHERE revoked_at IS NULL`,
		`ALTER TABLE api_key_usage DROP CONSTRAINT IF EXISTS fk_api_key_usage_key`,
		`ALTER TABLE api_key_usage ADD CONSTRAINT fk_api_key_usage_key FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE`,
		`ALTER TABLE alert_rules DROP CONSTRAINT IF EXISTS fk_alert_rules_saved_query`,
		`ALTER TABLE alert_rules ADD CONSTRAINT fk_alert_rules_saved_query FOREIGN KEY (saved_query_id) REFERENCES saved_queries(id) ON DELETE RESTRICT`,
		`ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS provider text NOT NULL DEFAULT 'webhook'`,
		`UPDATE alert_rules SET provider = 'webhook' WHERE provider IS NULL OR provider = ''`,
		`CREATE INDEX IF NOT EXISTS idx_alert_rules_provider ON alert_rules (provider) WHERE enabled`,
		`CREATE TABLE IF NOT EXISTS app_settings (id int PRIMARY KEY, antenne_url text NOT NULL DEFAULT '', antenne_secret text NOT NULL DEFAULT '', antenne_enabled bool NOT NULL DEFAULT false)`,
		convertLegacyOutbox,
		antenneOutbox,
		mcpkit.PendingSchema,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

// dropLocalAvatarCopies clears every avatar_url that is not an https URL.
//
// Journal used to download the provider's picture at sign-in and store the
// path of its own copy, /avatars/<key>. It now stores the provider's URL and no
// longer serves /avatars at all, so a row still pointing there is a broken
// image on every page. Clearing it shows the initial until the next sign-in
// writes the provider's stable URL. It matches what the sign-in itself stores,
// which is never anything but https or empty, so it is a no-op on every boot
// after the first.
const dropLocalAvatarCopies = `UPDATE users SET avatar_url = '' WHERE avatar_url <> '' AND avatar_url NOT ILIKE 'https://%'`

// porteSchema is porte/pg's Schema with its porte_users table left out and
// every foreign key repointed at Journal's own users.
//
// porte offers UserStore as the escape hatch for exactly this: users is
// referenced by name across this codebase and carries is_admin, which porte
// has no opinion about, so Journal keeps the table and implements the one
// method porte needs to resolve a callback to a row in it. The other three
// stores come from porte/pg unchanged — they only ever touch the tables below.
//
// Kept verbatim from porte otherwise, statement for statement: pg's queries are
// written against these names and a divergence here would surface as a runtime
// error on the login path rather than at boot. That includes porte v0.3.0's
// re-key of a password identity onto the account id — this app never calls
// portepg.EnsureSchema, so a migration living only in porte's own Schema would
// never run here and every existing password login would answer "invalid email
// or password" to the right password.
const porteSchema = `
CREATE TABLE IF NOT EXISTS porte_identities (
	user_id         bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	provider        text NOT NULL,
	subject         text NOT NULL,
	password_hash   text NOT NULL DEFAULT '',
	access_token    text NOT NULL DEFAULT '',
	refresh_token   text NOT NULL DEFAULT '',
	token_expiry    timestamptz,
	roles           jsonb,
	roles_synced_at timestamptz,
	synced_at       timestamptz,
	created_at      timestamptz DEFAULT now(),
	PRIMARY KEY (provider, subject)
);
CREATE INDEX IF NOT EXISTS porte_identities_user_idx ON porte_identities (user_id);
ALTER TABLE porte_identities ADD COLUMN IF NOT EXISTS created_at timestamptz;
ALTER TABLE porte_identities ALTER COLUMN created_at SET DEFAULT now();

UPDATE porte_identities SET subject = user_id::text
 WHERE provider = 'local' AND subject <> user_id::text;

CREATE TABLE IF NOT EXISTS porte_sessions (
	id           bigserial PRIMARY KEY,
	token_hash   text NOT NULL UNIQUE,
	user_id      bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	label        text NOT NULL DEFAULT '',
	created_at   timestamptz NOT NULL DEFAULT now(),
	last_used_at timestamptz NOT NULL DEFAULT now(),
	expires_at   timestamptz
);
CREATE INDEX IF NOT EXISTS porte_sessions_user_idx ON porte_sessions (user_id);
CREATE INDEX IF NOT EXISTS porte_sessions_expiry_idx ON porte_sessions (expires_at)
	WHERE expires_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS porte_login_codes (
	code_hash   text PRIMARY KEY,
	user_id     bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	expires_at  timestamptz NOT NULL,
	consumed_at timestamptz
);
`

// carryLegacySessionsOver moves the pre-porte sessions table across instead of
// dropping it, so adopting porte does not sign every existing user out.
//
// It works because the two tables agree on the only thing that matters: both
// store the SHA-256 hex of a 32-byte token and never the token itself, so a
// row copied here keeps authenticating the credential already in someone's
// browser. last_used_at is stamped now rather than copied from created_at —
// porte retires a browser session idle for seven days, and the old table
// recorded no use at all, so carrying created_at over would log out everyone
// who signed in more than a week ago on the deploy that is meant to keep them.
//
// The whole thing is guarded on the table existing, which makes it a no-op on
// a fresh install and on every boot after the first. to_regclass is given an
// unqualified name so it resolves through search_path, like every other
// statement here — hardcoding public would make this silently skip on any
// deployment that puts the app in a schema of its own.
const carryLegacySessionsOver = `
DO $$
BEGIN
	IF to_regclass('sessions') IS NOT NULL THEN
		INSERT INTO porte_sessions (token_hash, user_id, created_at, last_used_at, expires_at)
		SELECT s.token, s.user_id, s.created_at, now(), s.expires_at
		  FROM sessions s
		  JOIN users u ON u.id = s.user_id
		 WHERE s.expires_at > now()
		ON CONFLICT (token_hash) DO NOTHING;
		DROP TABLE sessions;
	END IF;
END
$$;
`

// adoptExistingPasswords moves the argon2 hashes from users.password_hash into
// the identity rows porte/local reads.
//
// Without it the v0.2 deploy silently ends password login for every existing
// account: the hash is still in the users table, nothing reads it there any
// more, and the login form answers "invalid email or password" to a correct
// password. The hashes are byte-identical — porte/local uses the parameters
// this app already used — so the move is a copy and nobody resets anything.
//
// The subject is the account id, which is what porte.LocalSubject returns since
// v0.3.0. It used to be the address, and this statement runs after the re-key
// above on every boot: writing the address here would insert a fresh
// address-keyed row immediately after the migration had finished removing them,
// with the stale hash from the users table.
//
// The source column is deliberately left in place. Blanking it in the same
// deploy would make the change unrollbackable for the sake of tidiness, and a
// column nothing reads can be dropped on any later day.
const adoptExistingPasswords = `
INSERT INTO porte_identities (user_id, provider, subject, password_hash)
SELECT id, 'local', id::text, password_hash
  FROM users
 WHERE password_hash <> ''
ON CONFLICT (provider, subject) DO NOTHING;
`

// antenneOutbox is antenne-client's outbox.Migration, Up half, made idempotent
// because Migrate runs on every boot. outbox_test.go proves the table it
// builds is the one the library's own migration builds.
const antenneOutbox = `
CREATE TABLE IF NOT EXISTS antenne_outbox (
    id BIGSERIAL PRIMARY KEY,
    channel TEXT NOT NULL,
    payload JSONB NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS antenne_outbox_unsent ON antenne_outbox (id) WHERE sent_at IS NULL;
CREATE INDEX IF NOT EXISTS antenne_outbox_sent ON antenne_outbox (sent_at) WHERE sent_at IS NOT NULL;
`

// convertLegacyOutbox turns the antenne_outbox Journal created before
// antenne-client v1 into the library's table, in place, so there is one
// outbox and not two. The old table carried attempts and last_error and no
// idempotency key; nothing ever wrote to it, but a row that is there keeps
// its id and gets a key of its own. A payload that is not JSON is kept as a
// JSON string rather than failing the boot. The attempts column is the
// marker, so every boot after the first skips this. Every step tolerates a
// half-converted table, because rolling back to v0.2 runs the old
// AutoMigrate over the new one: it adds attempts back and turns payload into
// text again, and the next boot of this version converts it once more.
const convertLegacyOutbox = `
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM information_schema.columns
	            WHERE table_schema = current_schema() AND table_name = 'antenne_outbox' AND column_name = 'attempts') THEN
		DROP INDEX IF EXISTS idx_antenne_outbox_created_at;
		ALTER TABLE antenne_outbox DROP COLUMN attempts, DROP COLUMN IF EXISTS last_error,
			ADD COLUMN IF NOT EXISTS idempotency_key text, ADD COLUMN IF NOT EXISTS sent_at timestamptz,
			DROP CONSTRAINT IF EXISTS antenne_outbox_idempotency_key_key;
		UPDATE antenne_outbox SET idempotency_key = coalesce(idempotency_key, 'journal_legacy_outbox_' || id),
			channel = coalesce(channel, ''), created_at = coalesce(created_at, now());
		ALTER TABLE antenne_outbox
			ALTER COLUMN payload TYPE jsonb
				USING CASE WHEN payload::text IS JSON THEN payload::text::jsonb ELSE to_jsonb(payload::text) END,
			ALTER COLUMN payload SET NOT NULL,
			ALTER COLUMN channel SET NOT NULL,
			ALTER COLUMN idempotency_key SET NOT NULL,
			ALTER COLUMN created_at SET DEFAULT now(),
			ALTER COLUMN created_at SET NOT NULL,
			ADD CONSTRAINT antenne_outbox_idempotency_key_key UNIQUE (idempotency_key);
	END IF;
END
$$;
`
