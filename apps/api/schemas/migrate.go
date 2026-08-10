package schemas

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&LogEntry{}, &User{}, &APIKey{}, &APIKeyUsage{}, &SavedQuery{}, &AlertRule{}); err != nil {
		return err
	}

	statements := []string{
		porteSchema,
		carryLegacySessionsOver,
		adoptExistingPasswords,
		`UPDATE users SET avatar_url = '' WHERE avatar_url IS NULL`,
		`ALTER TABLE users ALTER COLUMN avatar_url SET DEFAULT ''`,
		`ALTER TABLE users ALTER COLUMN avatar_url SET NOT NULL`,
		`ALTER TABLE log_entries ADD COLUMN IF NOT EXISTS search tsvector GENERATED ALWAYS AS (to_tsvector('simple', coalesce(message, ''))) STORED`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_search ON log_entries USING GIN(search)`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_app_created_at ON log_entries (app, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_created_at_id ON log_entries (created_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_log_entries_meta_request_id ON log_entries ((meta->>'request_id')) WHERE meta ? 'request_id'`,
		// Every key that predates the browser endpoint is a server
		// credential. Saying so explicitly matters more than it looks:
		// an empty kind would fall through the secret check and the
		// public check alike, and a key that authenticates nothing is a
		// silent outage on the next deploy.
		`UPDATE api_keys SET kind = 'secret' WHERE kind IS NULL OR kind = ''`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_kind ON api_keys (kind) WHERE revoked_at IS NULL`,
		`ALTER TABLE api_key_usage DROP CONSTRAINT IF EXISTS fk_api_key_usage_key`,
		`ALTER TABLE api_key_usage ADD CONSTRAINT fk_api_key_usage_key FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE`,
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

// porteSchema is porte/pg's Schema with its porte_users table left out and
// every foreign key repointed at Journal's own users.
//
// porte offers UserStore as the escape hatch for exactly this: users is
// referenced by name across this codebase and carries is_admin, which porte
// has no opinion about, so Journal keeps the table and implements the one
// method porte needs to resolve a callback to a row in it. The other three
// stores come from porte/pg unchanged — they only ever touch the tables below.
//
// Kept verbatim from porte otherwise, column for column: pg's queries are
// written against these names and a divergence here would surface as a runtime
// error on the login path rather than at boot.
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
// The source column is deliberately left in place. Blanking it in the same
// deploy would make the change unrollbackable for the sake of tidiness, and a
// column nothing reads can be dropped on any later day.
const adoptExistingPasswords = `
INSERT INTO porte_identities (user_id, provider, subject, password_hash)
SELECT id, 'local', email, password_hash
  FROM users
 WHERE password_hash <> ''
ON CONFLICT (provider, subject) DO NOTHING;
`
